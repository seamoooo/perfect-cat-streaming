package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	// nrmysql wraps go-sql-driver/mysql so DB queries become NR datastore segments
	// when the context carries a New Relic transaction.
	_ "github.com/newrelic/go-agent/v3/integrations/nrmysql"

	"github.com/seamoooo/perfect-cat-streaming/backend/internal/domain"
)

type MySQL struct {
	db *sql.DB
}

// DB exposes the raw *sql.DB for components that need direct access (e.g. the
// chaos package's SELECT SLEEP() bursts). Treat with care — the rest of the
// app should go through Repository methods.
func (m *MySQL) DB() *sql.DB { return m.db }

func NewMySQL(ctx context.Context, dsn string) (*MySQL, error) {
	driverDSN, err := normalizeMySQLDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("mysql dsn: %w", err)
	}
	db, err := sql.Open("nrmysql", driverDSN)
	if err != nil {
		return nil, fmt.Errorf("mysql open: %w", err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(30 * time.Minute)

	// Local container may still be starting; retry briefly.
	pingCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	var pingErr error
	for i := 0; i < 30; i++ {
		if pingErr = db.PingContext(pingCtx); pingErr == nil {
			break
		}
		select {
		case <-pingCtx.Done():
			return nil, fmt.Errorf("mysql ping: %w", pingCtx.Err())
		case <-time.After(time.Second):
		}
	}
	if pingErr != nil {
		return nil, fmt.Errorf("mysql ping: %w", pingErr)
	}

	m := &MySQL{db: db}
	// Ensure the schema exists. This mirrors infra/schema/schema.sql so a fresh
	// RDS instance (private subnet, unreachable from a laptop) becomes usable on
	// first boot with no manual `mysqldef` step. sqldef remains the source of
	// truth for richer changes locally; this is the idempotent prod bootstrap.
	if err := m.ensureSchema(ctx); err != nil {
		return nil, fmt.Errorf("mysql schema ensure: %w", err)
	}
	if err := m.seedIfEmpty(ctx); err != nil {
		return nil, fmt.Errorf("mysql seed: %w (table exists but seeding failed — likely permission or schema drift)", err)
	}
	return m, nil
}

// ensureSchema creates the `videos` table if it doesn't exist. Kept in sync
// with infra/schema/schema.sql (the declarative source of truth applied by
// mysqldef for non-trivial migrations). CREATE TABLE IF NOT EXISTS is a no-op
// when the table is already present, so this is safe to run on every startup.
func (m *MySQL) ensureSchema(ctx context.Context) error {
	const ddl = `
CREATE TABLE IF NOT EXISTS ` + "`videos`" + ` (
    ` + "`id`" + `           VARCHAR(64)  NOT NULL,
    ` + "`title`" + `        VARCHAR(255) NOT NULL,
    ` + "`description`" + `  TEXT         NOT NULL,
    ` + "`cat_name`" + `     VARCHAR(64)  NOT NULL,
    ` + "`breed`" + `        VARCHAR(32)  NOT NULL,
    ` + "`tags`" + `         JSON         NOT NULL,
    ` + "`duration_sec`" + ` DOUBLE       NOT NULL DEFAULT 0,
    ` + "`status`" + `       VARCHAR(32)  NOT NULL,
    ` + "`error_msg`" + `    TEXT         NOT NULL,
    ` + "`playlist_url`" + ` TEXT         NOT NULL,
    ` + "`created_at`" + `   DATETIME(6)  NOT NULL,
    ` + "`updated_at`" + `   DATETIME(6)  NOT NULL,
    PRIMARY KEY (` + "`id`" + `),
    INDEX ` + "`videos_created_at_idx`" + ` (` + "`created_at`" + `),
    INDEX ` + "`videos_status_idx`" + `     (` + "`status`" + `)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;`
	if _, err := m.db.ExecContext(ctx, ddl); err != nil {
		return err
	}
	return nil
}

// normalizeMySQLDSN converts the URL form "mysql://user:pw@host:port/db?..."
// into the go-sql-driver native DSN "user:pw@tcp(host:port)/db?...".
// If the input doesn't start with "mysql://", it's returned as-is so users can
// pass the driver DSN directly if they prefer.
func normalizeMySQLDSN(s string) (string, error) {
	if !strings.HasPrefix(s, "mysql://") {
		return s, nil
	}
	u, err := url.Parse(s)
	if err != nil {
		return "", err
	}
	pw, _ := u.User.Password()
	host := u.Host
	if !strings.Contains(host, ":") {
		host = host + ":3306"
	}
	dbname := strings.TrimPrefix(u.Path, "/")
	q := u.Query()
	if !q.Has("parseTime") {
		q.Set("parseTime", "true")
	}
	return fmt.Sprintf("%s:%s@tcp(%s)/%s?%s", u.User.Username(), pw, host, dbname, q.Encode()), nil
}

func (m *MySQL) seedIfEmpty(ctx context.Context) error {
	var n int
	if err := m.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM videos`).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	now := time.Now().UTC()
	seeds := []domain.Video{
		{
			ID: "welcome-bincho", Title: "ようこそ、Binchoの部屋へ",
			Description: "シャムのBinchoがお出迎え。MP4をアップロードすると本物のクリップが並びます。",
			CatName:     "Bincho", Breed: domain.BreedSiamese,
			Tags:      []string{"siamese", "welcome"},
			Status:    domain.StatusPending,
			CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "welcome-kanpachi", Title: "Kanpachi、獲物を狙う",
			Description: "ベンガルのKanpachiが計装済みプレイヤーで動きを観察します。",
			CatName:     "Kanpachi", Breed: domain.BreedBengal,
			Tags:      []string{"bengal", "welcome"},
			Status:    domain.StatusPending,
			CreatedAt: now, UpdatedAt: now,
		},
	}
	for _, v := range seeds {
		if err := m.Create(ctx, v); err != nil {
			return err
		}
	}
	return nil
}

func (m *MySQL) Create(ctx context.Context, v domain.Video) error {
	tags, err := json.Marshal(v.Tags)
	if err != nil {
		return err
	}
	_, err = m.db.ExecContext(ctx, `
INSERT INTO videos (id, title, description, cat_name, breed, tags, duration_sec, status, error_msg, playlist_url, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		v.ID, v.Title, v.Description, v.CatName, string(v.Breed), tags, v.DurationSec,
		string(v.Status), v.ErrorMsg, v.PlaylistURL, v.CreatedAt, v.UpdatedAt,
	)
	return err
}

func (m *MySQL) Get(ctx context.Context, id string) (domain.Video, bool) {
	row := m.db.QueryRowContext(ctx, `
SELECT id, title, description, cat_name, breed, tags, duration_sec, status, error_msg, playlist_url, created_at, updated_at
FROM videos WHERE id = ?`, id)
	v, err := mysqlScan(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Video{}, false
		}
		return domain.Video{}, false
	}
	return v, true
}

func (m *MySQL) List(ctx context.Context) []domain.Video {
	rows, err := m.db.QueryContext(ctx, `
SELECT id, title, description, cat_name, breed, tags, duration_sec, status, error_msg, playlist_url, created_at, updated_at
FROM videos ORDER BY created_at DESC`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := []domain.Video{}
	for rows.Next() {
		v, err := mysqlScan(rows)
		if err != nil {
			continue
		}
		out = append(out, v)
	}
	return out
}

func (m *MySQL) UpdateStatus(ctx context.Context, id string, status domain.Status, errMsg string) error {
	_, err := m.db.ExecContext(ctx, `
UPDATE videos SET status = ?, error_msg = ?, updated_at = ? WHERE id = ?`,
		string(status), errMsg, time.Now().UTC(), id)
	return err
}

func (m *MySQL) UpdateTags(ctx context.Context, id string, tags []string) error {
	if tags == nil {
		tags = []string{}
	}
	b, err := json.Marshal(tags)
	if err != nil {
		return err
	}
	_, err = m.db.ExecContext(ctx, `
UPDATE videos SET tags = ?, updated_at = ? WHERE id = ?`,
		b, time.Now().UTC(), id)
	return err
}

func (m *MySQL) UpdateMeta(ctx context.Context, id, title, description string) error {
	_, err := m.db.ExecContext(ctx, `
UPDATE videos SET title = ?, description = ?, updated_at = ? WHERE id = ?`,
		title, description, time.Now().UTC(), id)
	return err
}

func (m *MySQL) Delete(ctx context.Context, id string) error {
	res, err := m.db.ExecContext(ctx, `DELETE FROM videos WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("not found")
	}
	return nil
}

func (m *MySQL) UpdateAfterTranscode(ctx context.Context, id string, durationSec float64, playlistURL string) error {
	_, err := m.db.ExecContext(ctx, `
UPDATE videos
SET duration_sec = ?, playlist_url = ?, status = ?, error_msg = '', updated_at = ?
WHERE id = ?`,
		durationSec, playlistURL, string(domain.StatusReady), time.Now().UTC(), id)
	return err
}

type rowScanner interface {
	Scan(dest ...any) error
}

func mysqlScan(s rowScanner) (domain.Video, error) {
	var v domain.Video
	var breed, status string
	var tagsBytes []byte
	if err := s.Scan(
		&v.ID, &v.Title, &v.Description, &v.CatName, &breed, &tagsBytes,
		&v.DurationSec, &status, &v.ErrorMsg, &v.PlaylistURL,
		&v.CreatedAt, &v.UpdatedAt,
	); err != nil {
		return v, err
	}
	v.Breed = domain.Breed(breed)
	v.Status = domain.Status(status)
	if len(tagsBytes) > 0 {
		_ = json.Unmarshal(tagsBytes, &v.Tags)
	}
	return v, nil
}

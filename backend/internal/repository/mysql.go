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

	_ "github.com/go-sql-driver/mysql"

	"github.com/seamoooo/perfect-cat-streaming/backend/internal/domain"
)

type MySQL struct {
	db *sql.DB
}

func NewMySQL(ctx context.Context, dsn string) (*MySQL, error) {
	driverDSN, err := normalizeMySQLDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("mysql dsn: %w", err)
	}
	db, err := sql.Open("mysql", driverDSN)
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
	// Schema is managed externally via mysqldef (see infra/schema/schema.sql).
	// We only verify the table exists so we fail fast on misconfiguration.
	if err := m.assertSchema(ctx); err != nil {
		return nil, fmt.Errorf("mysql schema check: %w (run `make schema-apply` first)", err)
	}
	if err := m.seedIfEmpty(ctx); err != nil {
		return nil, fmt.Errorf("mysql seed: %w", err)
	}
	return m, nil
}

// assertSchema sanity-checks that the expected tables exist. The actual DDL
// lives in infra/schema/schema.sql and is applied by `mysqldef`.
func (m *MySQL) assertSchema(ctx context.Context) error {
	var n int
	err := m.db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM information_schema.tables
WHERE table_schema = DATABASE() AND table_name = 'videos'
`).Scan(&n)
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("table 'videos' missing")
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
		if err := m.Create(v); err != nil {
			return err
		}
	}
	return nil
}

func (m *MySQL) Create(v domain.Video) error {
	tags, err := json.Marshal(v.Tags)
	if err != nil {
		return err
	}
	_, err = m.db.ExecContext(context.Background(), `
INSERT INTO videos (id, title, description, cat_name, breed, tags, duration_sec, status, error_msg, playlist_url, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		v.ID, v.Title, v.Description, v.CatName, string(v.Breed), tags, v.DurationSec,
		string(v.Status), v.ErrorMsg, v.PlaylistURL, v.CreatedAt, v.UpdatedAt,
	)
	return err
}

func (m *MySQL) Get(id string) (domain.Video, bool) {
	row := m.db.QueryRowContext(context.Background(), `
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

func (m *MySQL) List() []domain.Video {
	rows, err := m.db.QueryContext(context.Background(), `
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

func (m *MySQL) UpdateStatus(id string, status domain.Status, errMsg string) error {
	_, err := m.db.ExecContext(context.Background(), `
UPDATE videos SET status = ?, error_msg = ?, updated_at = ? WHERE id = ?`,
		string(status), errMsg, time.Now().UTC(), id)
	return err
}

func (m *MySQL) UpdateAfterTranscode(id string, durationSec float64, playlistURL string) error {
	_, err := m.db.ExecContext(context.Background(), `
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

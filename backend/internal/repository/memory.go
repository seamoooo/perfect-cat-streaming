package repository

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/seamoooo/perfect-cat-streaming/backend/internal/domain"
)

var _ = context.Background // ctx-aware interface; Memory ignores ctx

// Memory is an in-memory repo with optional JSON persistence at filePath.
// Adequate for MVP; replace with a real DB later by satisfying repository.Repository.
type Memory struct {
	mu       sync.RWMutex
	items    map[string]domain.Video
	filePath string
}

func NewMemory(filePath string) (*Memory, error) {
	m := &Memory{items: map[string]domain.Video{}, filePath: filePath}
	if filePath != "" {
		if err := m.load(); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		m.seedIfEmpty()
	}
	return m, nil
}

func (m *Memory) load() error {
	b, err := os.ReadFile(m.filePath)
	if err != nil {
		return err
	}
	var arr []domain.Video
	if err := json.Unmarshal(b, &arr); err != nil {
		return err
	}
	for _, v := range arr {
		m.items[v.ID] = v
	}
	return nil
}

func (m *Memory) flush() error {
	if m.filePath == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(m.filePath), 0o755); err != nil {
		return err
	}
	arr := make([]domain.Video, 0, len(m.items))
	for _, v := range m.items {
		arr = append(arr, v)
	}
	b, err := json.MarshalIndent(arr, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.filePath, b, 0o644)
}

// seedIfEmpty writes welcome entries so the gallery is not empty on first boot.
// These are placeholder rows: status=pending and no actual media. They get
// real entries once a user uploads.
func (m *Memory) seedIfEmpty() {
	if len(m.items) > 0 {
		return
	}
	now := time.Now().UTC()
	seed := []domain.Video{
		{
			ID:          "welcome-bincho",
			Title:       "ようこそ、Binchoの部屋へ",
			Description: "シャムのBinchoがお出迎え。MP4をアップロードすると本物のクリップが並びます。",
			CatName:     "Bincho",
			Breed:       domain.BreedSiamese,
			Tags:        []string{"siamese", "welcome"},
			Status:      domain.StatusPending,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "welcome-kanpachi",
			Title:       "Kanpachi、獲物を狙う",
			Description: "ベンガルのKanpachiが計装済みプレイヤーで動きを観察します。",
			CatName:     "Kanpachi",
			Breed:       domain.BreedBengal,
			Tags:        []string{"bengal", "welcome"},
			Status:      domain.StatusPending,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}
	for _, v := range seed {
		m.items[v.ID] = v
	}
	_ = m.flush()
}

func (m *Memory) Create(_ context.Context, v domain.Video) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.items[v.ID]; ok {
		return errors.New("video already exists")
	}
	m.items[v.ID] = v
	return m.flush()
}

func (m *Memory) Get(_ context.Context, id string) (domain.Video, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.items[id]
	return v, ok
}

func (m *Memory) List(_ context.Context) []domain.Video {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]domain.Video, 0, len(m.items))
	for _, v := range m.items {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}

func (m *Memory) UpdateStatus(_ context.Context, id string, status domain.Status, errMsg string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.items[id]
	if !ok {
		return errors.New("not found")
	}
	v.Status = status
	v.ErrorMsg = errMsg
	v.UpdatedAt = time.Now().UTC()
	m.items[id] = v
	return m.flush()
}

func (m *Memory) UpdateTags(_ context.Context, id string, tags []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.items[id]
	if !ok {
		return errors.New("not found")
	}
	if tags == nil {
		tags = []string{}
	}
	v.Tags = tags
	v.UpdatedAt = time.Now().UTC()
	m.items[id] = v
	return m.flush()
}

func (m *Memory) Delete(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.items[id]; !ok {
		return errors.New("not found")
	}
	delete(m.items, id)
	return m.flush()
}

func (m *Memory) UpdateAfterTranscode(_ context.Context, id string, durationSec float64, playlistURL string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.items[id]
	if !ok {
		return errors.New("not found")
	}
	v.DurationSec = durationSec
	v.PlaylistURL = playlistURL
	v.Status = domain.StatusReady
	v.ErrorMsg = ""
	v.UpdatedAt = time.Now().UTC()
	m.items[id] = v
	return m.flush()
}

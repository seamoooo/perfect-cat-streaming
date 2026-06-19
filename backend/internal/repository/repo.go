package repository

import (
	"context"

	"github.com/seamoooo/perfect-cat-streaming/backend/internal/domain"
)

type Repository interface {
	Create(ctx context.Context, v domain.Video) error
	Get(ctx context.Context, id string) (domain.Video, bool)
	List(ctx context.Context) []domain.Video
	UpdateStatus(ctx context.Context, id string, status domain.Status, errMsg string) error
	UpdateAfterTranscode(ctx context.Context, id string, durationSec float64, playlistURL string) error
	UpdateTags(ctx context.Context, id string, tags []string) error
	Delete(ctx context.Context, id string) error
}

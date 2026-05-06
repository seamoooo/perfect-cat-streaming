package repository

import "github.com/seamoooo/perfect-cat-streaming/backend/internal/domain"

type Repository interface {
	Create(v domain.Video) error
	Get(id string) (domain.Video, bool)
	List() []domain.Video
	UpdateStatus(id string, status domain.Status, errMsg string) error
	UpdateAfterTranscode(id string, durationSec float64, playlistURL string) error
}

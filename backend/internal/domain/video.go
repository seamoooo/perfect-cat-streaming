package domain

import "time"

type Status string

const (
	StatusPending    Status = "pending"
	StatusProcessing Status = "processing"
	StatusReady      Status = "ready"
	StatusError      Status = "error"
)

type Breed string

const (
	BreedSiamese Breed = "siamese" // Bincho
	BreedBengal  Breed = "bengal"  // Kanpachi
	BreedOther   Breed = "other"
)

type Video struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	CatName     string    `json:"catName"`
	Breed       Breed     `json:"breed"`
	Tags        []string  `json:"tags"`
	DurationSec float64   `json:"durationSec"`
	Status      Status    `json:"status"`
	ErrorMsg    string    `json:"errorMsg,omitempty"`
	PlaylistURL string    `json:"playlistUrl,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

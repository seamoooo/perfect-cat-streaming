package storage

// Storage abstracts the path layout for cat clips. The current implementation is
// a local filesystem one (local.go), but a future S3-backed implementation can
// satisfy the same interface so the rest of the app stays unchanged.
type Storage interface {
	UploadPath(videoID, originalName string) string
	HLSDir(videoID string) string
	PlaylistPath(videoID string) string
	EnsureDirs() error
}

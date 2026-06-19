package publisher

import "context"

// Publisher uploads a transcoded HLS directory to a public origin (S3 + CloudFront)
// and returns the public playlist URL. When nil (local mode), the transcoder keeps
// serving from local disk via the /media route.
type Publisher interface {
	PublishHLS(ctx context.Context, localDir string, videoID string) (playlistURL string, err error)
	// DeleteHLS removes every object under <prefix>/<videoID>/ from the
	// origin. No-op if there's nothing there.
	DeleteHLS(ctx context.Context, videoID string) error
}

package publisher

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Config struct {
	Bucket    string // S3 bucket name (private, accessed via CloudFront OAC)
	Prefix    string // key prefix, e.g. "hls"
	Region    string // optional; falls back to default config chain
	MediaBase string // e.g. https://d1234.cloudfront.net (no trailing slash)
}

type S3 struct {
	cfg      S3Config
	uploader *manager.Uploader
}

func NewS3(ctx context.Context, c S3Config) (*S3, error) {
	if c.Bucket == "" {
		return nil, fmt.Errorf("publisher.NewS3: empty bucket")
	}
	if c.MediaBase == "" {
		return nil, fmt.Errorf("publisher.NewS3: empty MediaBase (CloudFront URL)")
	}
	c.Prefix = strings.Trim(c.Prefix, "/")
	c.MediaBase = strings.TrimRight(c.MediaBase, "/")

	loadOpts := []func(*awscfg.LoadOptions) error{}
	if c.Region != "" {
		loadOpts = append(loadOpts, awscfg.WithRegion(c.Region))
	}
	awsCfg, err := awscfg.LoadDefaultConfig(ctx, loadOpts...)
	if err != nil {
		return nil, fmt.Errorf("publisher.NewS3: load aws config: %w", err)
	}
	client := s3.NewFromConfig(awsCfg)
	return &S3{cfg: c, uploader: manager.NewUploader(client)}, nil
}

// PublishHLS uploads every file in localDir to s3://bucket/<prefix>/<videoID>/<filename>
// with appropriate Content-Type, then returns the CloudFront playlist URL.
func (p *S3) PublishHLS(ctx context.Context, localDir, videoID string) (string, error) {
	entries, err := os.ReadDir(localDir)
	if err != nil {
		return "", fmt.Errorf("read hls dir: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		full := filepath.Join(localDir, e.Name())
		f, err := os.Open(full)
		if err != nil {
			return "", fmt.Errorf("open %s: %w", full, err)
		}
		key := joinKey(p.cfg.Prefix, videoID, e.Name())
		ct := contentTypeFor(e.Name())
		_, uerr := p.uploader.Upload(ctx, &s3.PutObjectInput{
			Bucket:       &p.cfg.Bucket,
			Key:          &key,
			Body:         f,
			ContentType:  &ct,
			CacheControl: cacheControlFor(e.Name()),
		})
		f.Close()
		if uerr != nil {
			return "", fmt.Errorf("upload %s: %w", key, uerr)
		}
	}

	playlistKey := joinKey(p.cfg.Prefix, videoID, "index.m3u8")
	return p.cfg.MediaBase + "/" + playlistKey, nil
}

func joinKey(parts ...string) string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.Trim(p, "/")
		if p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, "/")
}

func contentTypeFor(name string) string {
	switch {
	case strings.HasSuffix(name, ".m3u8"):
		return "application/vnd.apple.mpegurl"
	case strings.HasSuffix(name, ".ts"):
		return "video/mp2t"
	case strings.HasSuffix(name, ".vtt"):
		return "text/vtt"
	default:
		return "application/octet-stream"
	}
}

func cacheControlFor(name string) *string {
	// Playlists need to refresh for live updates; segments are immutable.
	var v string
	if strings.HasSuffix(name, ".m3u8") {
		v = "public, max-age=10"
	} else {
		v = "public, max-age=31536000, immutable"
	}
	return &v
}

package config

import (
	"os"
	"strings"
)

type Config struct {
	AppPort        string
	UploadDir      string
	HLSDir         string
	PublicBaseURL  string
	AllowedOrigins []string
	FFmpegBin      string

	// S3 publishing (optional). When S3Bucket is non-empty the transcoder
	// uploads HLS to S3 after each job and stores the CloudFront playlist URL
	// on the Video. Otherwise the local /media route serves the files.
	S3Bucket    string
	S3Prefix    string
	S3Region    string
	MediaBaseURL string // e.g. https://d1234.cloudfront.net
}

func Load() Config {
	return Config{
		AppPort:        getenv("APP_PORT", "8080"),
		UploadDir:      getenv("STORAGE_UPLOAD_DIR", "/var/app/data/uploads"),
		HLSDir:         getenv("STORAGE_HLS_DIR", "/var/app/data/hls"),
		PublicBaseURL:  getenv("PUBLIC_BASE_URL", "http://localhost:8080"),
		AllowedOrigins: splitCSV(getenv("ALLOWED_ORIGINS", "http://localhost:5173,http://localhost:8000")),
		FFmpegBin:      getenv("FFMPEG_BIN", "ffmpeg"),
		S3Bucket:       getenv("S3_BUCKET", ""),
		S3Prefix:       getenv("S3_HLS_PREFIX", "hls"),
		S3Region:       getenv("S3_REGION", ""),
		MediaBaseURL:   getenv("MEDIA_BASE_URL", ""),
	}
}

func getenv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

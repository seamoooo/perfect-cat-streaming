package tests

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/seamoooo/perfect-cat-streaming/backend/internal/config"
	httpx "github.com/seamoooo/perfect-cat-streaming/backend/internal/http"
	"github.com/seamoooo/perfect-cat-streaming/backend/internal/repository"
	"github.com/seamoooo/perfect-cat-streaming/backend/internal/storage"
	"github.com/seamoooo/perfect-cat-streaming/backend/internal/transcoder"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	tmp := t.TempDir()
	cfg := config.Config{
		AppPort:        "0",
		UploadDir:      tmp + "/uploads",
		HLSDir:         tmp + "/hls",
		PublicBaseURL:  "http://test",
		AllowedOrigins: []string{"http://test"},
		FFmpegBin:      "ffmpeg",
	}
	stg := storage.NewLocal(cfg.UploadDir, cfg.HLSDir)
	if err := stg.EnsureDirs(); err != nil {
		t.Fatal(err)
	}
	repo, err := repository.NewMemory("")
	if err != nil {
		t.Fatal(err)
	}
	tx := transcoder.New(cfg.FFmpegBin)
	q := transcoder.NewQueue(tx, repo, stg, nil, 1)
	return httptest.NewServer(httpx.NewRouter(cfg, repo, stg, q))
}

func TestHealthz(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()
	res, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 200 {
		t.Fatalf("want 200, got %d", res.StatusCode)
	}
}

func TestMeowReturnsKnownCat(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()
	res, err := http.Get(srv.URL + "/meow")
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 200 {
		t.Fatalf("want 200, got %d", res.StatusCode)
	}
	buf := make([]byte, 1024)
	n, _ := res.Body.Read(buf)
	body := string(buf[:n])
	if !strings.Contains(body, "Bincho") && !strings.Contains(body, "Kanpachi") {
		t.Fatalf("expected Bincho or Kanpachi in response, got %s", body)
	}
}

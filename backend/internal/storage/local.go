package storage

import (
	"os"
	"path/filepath"
)

type Local struct {
	UploadRoot string
	HLSRoot    string
}

func NewLocal(uploadRoot, hlsRoot string) *Local {
	return &Local{UploadRoot: uploadRoot, HLSRoot: hlsRoot}
}

func (l *Local) EnsureDirs() error {
	for _, p := range []string{l.UploadRoot, l.HLSRoot} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func (l *Local) UploadPath(videoID, originalName string) string {
	return filepath.Join(l.UploadRoot, videoID+"_"+filepath.Base(originalName))
}

func (l *Local) HLSDir(videoID string) string {
	return filepath.Join(l.HLSRoot, videoID)
}

func (l *Local) PlaylistPath(videoID string) string {
	return filepath.Join(l.HLSDir(videoID), "index.m3u8")
}

func (l *Local) RemoveVideo(videoID string) error {
	matches, _ := filepath.Glob(filepath.Join(l.UploadRoot, videoID+"_*"))
	for _, p := range matches {
		_ = os.Remove(p)
	}
	return os.RemoveAll(l.HLSDir(videoID))
}

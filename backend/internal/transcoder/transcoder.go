package transcoder

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type FFmpeg struct {
	Bin string
}

func New(bin string) *FFmpeg {
	if bin == "" {
		bin = "ffmpeg"
	}
	return &FFmpeg{Bin: bin}
}

// TranscodeToHLS converts srcMP4 into an HLS playlist + segments under outDir.
// Single-bitrate for the MVP; the front end's KanpachiPlayer can still observe
// kanpachi.stretch (bitrate change) once a multi-bitrate ladder is added later.
func (f *FFmpeg) TranscodeToHLS(ctx context.Context, srcMP4, outDir string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	args := []string{
		"-y",
		"-i", srcMP4,
		"-c:v", "libx264",
		"-preset", "veryfast",
		"-c:a", "aac",
		"-ac", "2",
		"-b:a", "128k",
		"-hls_time", "6",
		"-hls_list_size", "0",
		"-hls_segment_filename", outDir + "/seg_%05d.ts",
		"-f", "hls",
		outDir + "/index.m3u8",
	}
	cmd := exec.CommandContext(ctx, f.Bin, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg failed: %w: %s", err, stderr.String())
	}
	return nil
}

// Probe returns duration in seconds via ffprobe (bundled with ffmpeg).
func (f *FFmpeg) Probe(ctx context.Context, srcMP4 string) (float64, error) {
	probe := strings.TrimSuffix(f.Bin, "ffmpeg") + "ffprobe"
	cmd := exec.CommandContext(ctx, probe,
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		srcMP4,
	)
	out, err := cmd.Output()
	if err != nil {
		return 0, errors.New("ffprobe failed: " + err.Error())
	}
	d, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	if err != nil {
		return 0, err
	}
	return d, nil
}

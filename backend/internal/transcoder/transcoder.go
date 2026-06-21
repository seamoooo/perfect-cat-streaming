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

	"github.com/newrelic/go-agent/v3/newrelic"
)

// segment starts a NR custom segment if a transaction is in ctx; otherwise the
// returned func is a no-op.
func segment(ctx context.Context, name string) func() {
	if txn := newrelic.FromContext(ctx); txn != nil {
		s := txn.StartSegment(name)
		return func() { s.End() }
	}
	return func() {}
}

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
//
// When slow is true the developer "SRE" demo is active: we read the input at its
// native frame rate (-re) and use the slowest x264 preset, which collapses
// transcode throughput so transcode.realtime_factor jumps from ~0.1 to ~1.0+.
// This is a deliberate, clearly-labelled degradation for New Relic incident
// drills — never enabled by ordinary uploads.
func (f *FFmpeg) TranscodeToHLS(ctx context.Context, srcMP4, outDir string, slow bool) error {
	defer segment(ctx, "ffmpeg.transcode")()
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	preset := "veryfast"
	args := []string{"-y"}
	if slow {
		preset = "veryslow"
		// -re must precede -i: throttle reading to real time, killing throughput.
		args = append(args, "-re")
	}
	args = append(args,
		"-i", srcMP4,
		"-c:v", "libx264",
		"-preset", preset,
		"-c:a", "aac",
		"-ac", "2",
		"-b:a", "128k",
		"-hls_time", "6",
		"-hls_list_size", "0",
		"-hls_segment_filename", outDir+"/seg_%05d.ts",
		"-f", "hls",
		outDir+"/index.m3u8",
	)
	cmd := exec.CommandContext(ctx, f.Bin, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg failed: %w: %s", err, stderr.String())
	}
	return nil
}

// Thumbnail extracts a single representative frame from srcMP4 and writes it as
// a JPEG to outPath — used as the Netflix-style poster image in the gallery.
// The `thumbnail` filter scans frames and picks the most representative one,
// so we avoid an all-black first frame.
func (f *FFmpeg) Thumbnail(ctx context.Context, srcMP4, outPath string) error {
	defer segment(ctx, "ffmpeg.thumbnail")()
	args := []string{
		"-y",
		"-i", srcMP4,
		"-vf", "thumbnail,scale=640:-2",
		"-frames:v", "1",
		outPath,
	}
	cmd := exec.CommandContext(ctx, f.Bin, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg thumbnail failed: %w: %s", err, stderr.String())
	}
	return nil
}

// PosterFromImage normalizes a user-supplied image (jpg/png/webp) into the
// poster.jpg used by the gallery — scaled to a consistent width and re-encoded
// as JPEG. Used when the uploader picks a custom thumbnail.
func (f *FFmpeg) PosterFromImage(ctx context.Context, srcImage, outPath string) error {
	defer segment(ctx, "ffmpeg.poster_image")()
	args := []string{
		"-y",
		"-i", srcImage,
		"-vf", "scale=640:-2",
		"-frames:v", "1",
		outPath,
	}
	cmd := exec.CommandContext(ctx, f.Bin, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg poster-from-image failed: %w: %s", err, stderr.String())
	}
	return nil
}

// Probe returns duration in seconds via ffprobe (bundled with ffmpeg).
func (f *FFmpeg) Probe(ctx context.Context, srcMP4 string) (float64, error) {
	defer segment(ctx, "ffprobe.duration")()
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

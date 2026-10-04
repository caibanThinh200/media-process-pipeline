package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
)

// FFmpegTranscoder executes ffmpeg via command line.
type FFmpegTranscoder struct {
	binaryPath string
}

// NewFFmpegTranscoder discovers the ffmpeg executable.
// Checks /opt/bin/ffmpeg first (AWS Lambda layer path), then PATH.
func NewFFmpegTranscoder() *FFmpegTranscoder {
	bin := "/opt/bin/ffmpeg"
	if _, err := os.Stat(bin); err != nil {
		// Fallback to system PATH for local dev / testing
		if path, err := exec.LookPath("ffmpeg"); err == nil {
			bin = path
		} else {
			bin = "ffmpeg"
		}
	}
	log.Printf("FFmpeg binary path: %s", bin)
	return &FFmpegTranscoder{binaryPath: bin}
}

// Transcode converts the input video to a web-optimized 720p H.264 MP4.
// Arguments:
//   - -y: Overwrite output file
//   - -i: Input path
//   - -vf scale='min(1280,iw)':-2: Scale width up to 1280 maintaining aspect ratio (even dimensions)
//   - -c:v libx264: Standard H.264 video codec for browser compatibility
//   - -crf 23: Balanced quality and file size
//   - -preset fast: Fast encoding speed
//   - -c:a aac -b:a 128k: Stereo AAC audio
//   - -movflags +faststart: Relocate moov atom to beginning of file for instant web playback
func (t *FFmpegTranscoder) Transcode(ctx context.Context, inputPath, outputPath string) error {
	args := []string{
		"-y",
		"-i", inputPath,
		"-vf", "scale=min(1280\\,iw):-2",
		"-c:v", "libx264",
		"-crf", "23",
		"-preset", "fast",
		"-c:a", "aac",
		"-b:a", "128k",
		"-movflags", "+faststart",
		outputPath,
	}

	cmd := exec.CommandContext(ctx, t.binaryPath, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg transcode failed: %w (output: %s)", err, string(out))
	}
	return nil
}

// Copyright ©2022 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Ffmpeg family related tools.
package tools

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path"

	"github.com/evolution-gaming/ease/internal/logging"
	"github.com/evolution-gaming/ease/internal/video"
)

var (
	ffprobeCmd         = "ffprobe"
	ffprobeEnvOverride = "FFPROBE_EXE_PATH"
	ffmpegCmd          = "ffmpeg"
	ffmpegEnvOverride  = "FFMPEG_EXE_PATH"
	// A specific libvmaf model file to be used when calculating VMAF score.
	libvmafModel            = "vmaf_v0.6.1.json"
	libvmafModelEnvOverride = "LIBVMAF_MODEL_PATH"
	// A list of known locations where various distributions of ffmpeg may put
	// libvmaf models.
	libvmafModelLocations = []string{
		"/usr/local/share/model",
		"/usr/share/model",
		"/opt/ffmpeg-static/model",
	}
)

// FfmpegPath will return path to ffmpeg binary and error if path is not found.
func FfmpegPath() (string, error) {
	p, err := FindTool(ffmpegCmd, ffmpegEnvOverride)
	if err != nil {
		return "", fmt.Errorf("ffmpeg not found: %w", err)
	}
	return p, nil
}

// FfprobePath will return path to ffprobe binary and error if path is not found.
func FfprobePath() (string, error) {
	p, err := FindTool(ffprobeCmd, ffprobeEnvOverride)
	if err != nil {
		return "", fmt.Errorf("ffprobe not found: %w", err)
	}
	return p, nil
}

// FfprobeExtractMetadata will query vide file metadata via ffprobe.
func FfprobeExtractMetadata(videoFile string) (video.Metadata, error) {
	var vmeta video.Metadata

	type metadata struct {
		CodecName string  `json:"codec_name,omitempty"`
		FrameRate string  `json:"r_frame_rate,omitempty"`
		Duration  float64 `json:"duration,omitempty,string"`
		Width     int     `json:"width,omitempty"`
		Height    int     `json:"height,omitempty"`
		BitRate   int     `json:"bit_rate,omitempty,string"`
	}

	if _, err := os.Stat(videoFile); os.IsNotExist(err) {
		return vmeta, fmt.Errorf("FfprobeExtractMetadata() os.Stat: %w", err)
	}

	ffprobeArgs := []string{
		"-v", "quiet",
		"-select_streams", "v",
		"-of", "json",
		"-show_format",
		"-show_streams",
		videoFile,
	}
	ffprobePath, err := FfprobePath()
	if err != nil {
		return vmeta, err
	}
	cmd := exec.Command(ffprobePath, ffprobeArgs...)
	logging.Debugf("Running: %s\n", cmd)
	out, err := cmd.Output()
	if err != nil {
		return vmeta, fmt.Errorf("FfprobeExtractMetadata() exec error: %w", err)
	}

	// Unmarshal metadata from both "streams" and "format" JSON objects.
	meta := &struct {
		Streams []metadata
		Format  metadata
	}{}

	if err := json.Unmarshal(out, &meta); err != nil {
		return vmeta, fmt.Errorf("FfprobeExtractMetadata() json.Unmarshal: %w", err)
	}
	logging.Debugf("%s %+v", videoFile, meta)
	vmeta = video.Metadata(meta.Streams[0])
	// For mkv container Streams does not contain duration, so we have to look into Format.
	vmeta.Duration = math.Max(vmeta.Duration, meta.Format.Duration)

	return vmeta, nil
}

// FindLibvmafModel will return path to libvmaf model file.
//
// XXX: Although not specifically related to ffmpeg family tools, but for time
// being keep it here.
func FindLibvmafModel() (string, error) {
	// First check for model in case it's overridden via env variable.
	if p := os.Getenv(libvmafModelEnvOverride); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	for _, l := range libvmafModelLocations {
		p := path.Join(l, libvmafModel)
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	return "", fmt.Errorf("libvmaf model file %s not found in any of %s", libvmafModel, libvmafModelLocations)
}

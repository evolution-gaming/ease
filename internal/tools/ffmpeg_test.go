// Copyright ©2022 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package tools

import (
	"os"
	"path"
	"testing"

	"github.com/evolution-gaming/ease/internal/video"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Path(t *testing.T) {
	type testCase struct {
		pathFunc func() (string, error)
		exeName  string
	}

	tests := map[string]testCase{
		"FfprobePath()": {
			pathFunc: FfprobePath,
			exeName:  "ffprobe",
		},
		"FfmpegPath()": {
			pathFunc: FfmpegPath,
			exeName:  "ffmpeg",
		},
	}

	run := func(t *testing.T, tc testCase) {
		// Create a fake binary and put it on PATH
		fakeBinDir := t.TempDir()
		wantPath := path.Join(fakeBinDir, tc.exeName)
		f, err := os.OpenFile(wantPath, os.O_CREATE, 0o755)
		require.NoError(t, err)
		f.Close()
		sysPath := os.Getenv("PATH")
		t.Setenv("PATH", fakeBinDir+":"+sysPath)

		gotPath, err := tc.pathFunc()
		assert.NoError(t, err)

		assert.Equal(t, wantPath, gotPath)
		assert.FileExists(t, gotPath)
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func Test_Path_Negative(t *testing.T) {
	type testCase struct {
		pathFunc func() (string, error)
	}

	tests := map[string]testCase{
		"FfprobePath()": {
			pathFunc: FfprobePath,
		},
		"FfmpegPath()": {
			pathFunc: FfmpegPath,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Wipe PATH so that no binary can be located.
			t.Setenv("PATH", "")

			s, err := tc.pathFunc()
			assert.Error(t, err, "Expected error since binary is not on PATH")
			assert.Equal(t, "", s, "Expected empty string as path")
		})
	}
}

func Test_FfprobeExtractMetadata(t *testing.T) {
	type testCase struct {
		videoFile    string
		wantMetadata video.Metadata
	}

	tests := map[string]testCase{
		"h264 mp4 small": {
			videoFile: "../../testdata/video/testsrc01.mp4",
			wantMetadata: video.Metadata{
				Duration:   1,
				Width:      320,
				Height:     240,
				BitRate:    56112,
				FrameCount: 10,
				CodecName:  "h264",
				FrameRate:  "10/1",
			},
		},
		"h264 mp4 large": {
			videoFile: "../../testdata/video/testsrc02.mp4",
			wantMetadata: video.Metadata{
				Duration:   10,
				Width:      1280,
				Height:     720,
				BitRate:    86740,
				FrameCount: 240,
				CodecName:  "h264",
				FrameRate:  "24/1",
			},
		},
		"av1 ivf": {
			videoFile: "../../testdata/video/testsrc03.ivf",
			wantMetadata: video.Metadata{
				Duration:   1,
				Width:      1920,
				Height:     1080,
				BitRate:    114648,
				FrameCount: 24,
				CodecName:  "av1",
				FrameRate:  "24/1",
			},
		},
		"av1 mp4": {
			videoFile: "../../testdata/video/testsrc04.mp4",
			wantMetadata: video.Metadata{
				Duration:   1,
				Width:      1920,
				Height:     1080,
				BitRate:    111704,
				FrameCount: 24,
				CodecName:  "av1",
				FrameRate:  "24/1",
			},
		},
		"h264 mkv": {
			videoFile: "../../testdata/video/testsrc05.mkv",
			wantMetadata: video.Metadata{
				Duration:   1,
				Width:      320,
				Height:     240,
				BitRate:    62648,
				FrameCount: 10,
				CodecName:  "h264",
				FrameRate:  "10/1",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			gotMetadata, err := FfprobeExtractMetadata(tc.videoFile)
			assert.NoError(t, err)
			assert.Equal(t, tc.wantMetadata, gotMetadata)
		})
	}
}

func Test_FfprobeExtractMetadata_Negative(t *testing.T) {
	t.Run("Should fail for non-existent media file", func(t *testing.T) {
		_, err := FfprobeExtractMetadata("/non/existent/path/to/file")
		assert.Error(t, err)
	})
	t.Run("Should fail extracting metadata from non-media file", func(t *testing.T) {
		// Try to extract metadata from non video file, just some binary like for instance
		// a test binary.
		nonMediaFile := os.Args[0]
		_, err := FfprobeExtractMetadata(nonMediaFile)
		assert.Error(t, err)
	})
}

func Test_FindLibvmafModel(t *testing.T) {
	t.Run("Model path should be valid", func(t *testing.T) {
		gotPath, err := FindLibvmafModel()
		assert.NoError(t, err)
		assert.FileExists(t, gotPath)
	})
}

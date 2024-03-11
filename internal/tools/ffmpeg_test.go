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
	videoFile := "../../testdata/video/testsrc02.mp4"
	t.Run("Should extract VideoMetadata from video file", func(t *testing.T) {
		want := video.Metadata{
			Duration:   10,
			Width:      1280,
			Height:     720,
			BitRate:    86740,
			FrameCount: 240,
			CodecName:  "h264",
			FrameRate:  "24/1",
		}

		got, err := FfprobeExtractMetadata(videoFile)
		assert.NoError(t, err)
		assert.Equal(t, want, got)
	})
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

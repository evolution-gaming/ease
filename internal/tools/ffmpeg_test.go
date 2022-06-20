// Copyright ©2022 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package tools

import (
	"os"
	"path"
	"testing"

	"github.com/evolution-gaming/ease/internal/video"
	"github.com/google/go-cmp/cmp"
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
		if err != nil {
			t.Fatal(err)
		}
		f.Close()
		sysPath := os.Getenv("PATH")
		t.Setenv("PATH", fakeBinDir+":"+sysPath)

		gotPath, err := tc.pathFunc()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if diff := cmp.Diff(wantPath, gotPath); diff != "" {
			t.Errorf("Path mismatch (-want +got):\n%s", diff)
		}

		if _, err := os.Stat(gotPath); err != nil {
			t.Errorf("%s's path does not exist: %v", tc.exeName, err)
		}
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
			if err == nil {
				t.Error("Expected error since binary is not on PATH, but got <nil>")
			}

			if s != "" {
				t.Errorf("Expected empty string as path, but got: %v", s)
			}
		})
	}
}

func Test_FfprobeExtractMetadata(t *testing.T) {
	videoFile := "../../testdata/video/testsrc02.mp4"
	t.Run("Should extract VideoMetadata from video file", func(t *testing.T) {
		want := video.Metadata{
			Duration:  10,
			Width:     1280,
			Height:    720,
			BitRate:   86740,
			CodecName: "h264",
			FrameRate: "24/1",
		}

		got, err := FfprobeExtractMetadata(videoFile)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("FfprobeExtractMetadata() mismatch (-want +got):\n%s", diff)
		}
	})
}

func Test_FfprobeExtractMetadata_Negative(t *testing.T) {
	t.Run("Should fail for non-existent media file", func(t *testing.T) {
		_, err := FfprobeExtractMetadata("/non/existent/path/to/file")
		if err == nil {
			t.Error("Expected error, but got <nil>")
		}
	})
	t.Run("Should fail extracting metadata from non-media file", func(t *testing.T) {
		// Try to extract metadata from non video file, just some binary like for instance
		// a test binary.
		nonMediaFile := os.Args[0]
		_, err := FfprobeExtractMetadata(nonMediaFile)
		if err == nil {
			t.Error("Expected error, but got <nil>")
		}
	})
}

func Test_FindLibvmafModel(t *testing.T) {
	t.Run("Model path should be valid", func(t *testing.T) {
		checkModelFile := func(t *testing.T, fPath string) {
			if _, err := os.Stat(fPath); err != nil {
				t.Errorf("Model file path is not valid: %v", err)
			}
		}

		gotPath, err := FindLibvmafModel()
		if err != nil {
			t.Errorf("Unexpected error locating libvmaf model file: %v", err)
		}
		checkModelFile(t, gotPath)
	})

	t.Run("Override via environment var", func(t *testing.T) {
		// Create a fake model file
		fakeModelFile := path.Join(t.TempDir(), libvmafModel)
		t.Setenv(libvmafModelEnvOverride, fakeModelFile)
		f, err := os.OpenFile(fakeModelFile, os.O_CREATE, 0o644)
		if err != nil {
			t.Fatal(err)
		}
		f.Close()

		gotPath, err := FindLibvmafModel()
		if err != nil {
			t.Errorf("Unexpected error locating model file: %v", err)
		}

		if diff := cmp.Diff(fakeModelFile, gotPath); diff != "" {
			t.Errorf("Model file path mismatch (-want +got):\n%s", diff)
		}
	})
}

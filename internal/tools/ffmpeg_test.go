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

func Test_FfprobePath(t *testing.T) {
	// Create a fake ffprobe binary and prepend it to PATH.
	fakeBinDir := t.TempDir()
	wantPath := path.Join(fakeBinDir, "ffprobe")
	f, err := os.OpenFile(wantPath, os.O_CREATE, 0o755)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	sysPath := os.Getenv("PATH")
	t.Setenv("PATH", fakeBinDir+":"+sysPath)

	t.Log("Call to Path() should result in no error")
	gotPath, err := FfprobePath()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	t.Log("Call to Path() should return ffprobe's path")
	if diff := cmp.Diff(wantPath, gotPath); diff != "" {
		t.Errorf("FfprobePath() mismatch (-want +got):\n%s", diff)
	}

	t.Logf("Executable path (%s) should exist", gotPath)
	if _, err := os.Stat(gotPath); err != nil {
		t.Errorf("ffprobe's path does not exist: %v", err)
	}
}

func Test_FfprobePath_Negative(t *testing.T) {
	// Wipe PATH so that no binary can be located including ffprobe.
	t.Setenv("PATH", "")

	t.Log("Call to FfprobePath() should result in error")
	s, err := FfprobePath()
	if err == nil {
		t.Error("Error expected, but got <nil>")
	}

	if s != "" {
		t.Errorf("Expected empty string as path, but got: %v", s)
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
	t.Run("Should fail for non-existent vide file", func(t *testing.T) {
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

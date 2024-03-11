// Copyright ©2022 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Reusable helpers and fixtures for tests.
package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"testing"

	"github.com/evolution-gaming/ease/internal/tools"
)

// fixPlanConfig fixture provides simple encoding plan.
//
// To make "encoding" faster we just copy source to destination, for the
// purposes of tests it is irrelevant if we use realistic encoder or just a
// simple file copy.
func fixPlanConfig(t *testing.T) (fPath string) {
	payload := []byte(`{
		"Inputs": [
			"testdata/video/testsrc01.mp4"
		],
		"Schemes": [
			{
				"Name": "simple_src_duplication",
				"CommandTpl": ["cp -v ",  "%INPUT% ", "%OUTPUT%.mp4"]
			}
		]
	}`)
	fPath = path.Join(t.TempDir(), "minimal.json")
	err := os.WriteFile(fPath, payload, fs.FileMode(0o644))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	return
}

// fixPlanConfigInvalid fixture provides invalid encoding plan.
func fixPlanConfigInvalid(t *testing.T) (fPath string) {
	payload := []byte(`{
		"Inputs": [
			"non-existent"
		]
	}`)
	fPath = path.Join(t.TempDir(), "minimal.json")
	err := os.WriteFile(fPath, payload, fs.FileMode(0o644))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	return fPath
}

// fixCreateFakeFfmpegAndPutItOnPath fixture create a fake and failing ffmpeg on PATH.
func fixCreateFakeFfmpegAndPutItOnPath(t *testing.T) {
	origPath := os.Getenv("PATH")
	fakePath := t.TempDir()
	t.Setenv("PATH", fmt.Sprintf("%s:%s", fakePath, origPath))

	// Make ffmpeg binary contain this helper contents, will just print to
	// stderr all provided command line args. This should be enough to trigger
	// failure for anything that requires proper function from ffmpeg.
	src, err := os.Open("testdata/helpers/stderr")
	if err != nil {
		t.Fatalf("Unable to open source: %v", err)
	}
	defer src.Close()
	dst, err := os.OpenFile(path.Join(fakePath, "ffmpeg"), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		t.Fatalf("Unable to open destination: %v", err)
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		t.Fatalf("Failure copying: %v", err)
	}
}

// fixPlanConfigMisalignedFrames fixture returns a plan which will result in encoded file
// shorter by 1 frame e.g. first frame dropped.
//
// Note: this plan assumes ffmpeg doing actual encoding!
func fixPlanConfigMisalignedFrames(t *testing.T) (fPath string) {
	ffmpegPath, err := tools.FfmpegPath()
	if err != nil {
		t.Fatalf("ffmpeg not found: %v", err)
	}
	payload := []byte(fmt.Sprintf(`{
		"Inputs": [
			"testdata/video/testsrc01.mp4"
		],
		"Schemes": [
			{
				"Name": "misaligned",
				"CommandTpl": ["%s -i %%INPUT%% -vf \"trim=start_frame=1\" %%OUTPUT%%.mp4"]
			}
		]
	}`, ffmpegPath))

	fPath = path.Join(t.TempDir(), "misaligned_plan.json")
	err = os.WriteFile(fPath, payload, fs.FileMode(0o644))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	return
}

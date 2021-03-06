// Copyright ©2022 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package vqm

import (
	"testing"

	"github.com/evolution-gaming/ease/internal/tools"
	"github.com/google/go-cmp/cmp"
)

func TestFfmpegVMAFImplementsMeasurer(t *testing.T) {
	// Test that tool implement Measurer interface.
	var _ Measurer = &ffmpegVMAF{}
}

func TestFfmpegVMAF(t *testing.T) {
	var tool Measurer // tool under test
	var result Result // result from tool under test

	wrkDir := t.TempDir()
	ffmpegExePath, _ := tools.FfmpegPath()
	libvmafModelPath, _ := tools.FindLibvmafModel()

	srcFile := "../../testdata/video/testsrc01.mp4"
	compressedFile := "../../testdata/video/testsrc01.mp4"
	resultFile := wrkDir + "/result.json"

	t.Run("NewFfmpegVMAF creates new VQM tool", func(t *testing.T) {
		var err error
		tool, err = NewFfmpegVMAF(ffmpegExePath, libvmafModelPath, compressedFile, srcFile, resultFile)
		if err != nil {
			t.Errorf("Unexpected error when calling NewFfmpegVMAF(): %v", err)
		}
	})

	t.Run("Call Measure()", func(t *testing.T) {
		err := tool.Measure()
		if err != nil {
			t.Errorf("Unexpected error calling Measure(): %v", err)
			vt, ok := tool.(*ffmpegVMAF)
			if ok {
				t.Logf("Tool path: %s\nWith output:\n%s", vt.exePath, vt.output)
			}
		}
	})

	t.Run("Call GetResult()", func(t *testing.T) {
		var err error
		result, err = tool.GetResult()
		if err != nil {
			t.Errorf("Unexpected error calling GetResult(): %v", err)
		}
		t.Logf("result: %v", result)
	})

	t.Run("VideoQualityResult should have metrics", func(t *testing.T) {
		if result.Metrics.PSNR == 0 {
			t.Errorf("No PSNR metric detected: %#v", result)
		}
		if result.Metrics.VMAF == 0 {
			t.Errorf("No VMAF metric detected: %#v", result)
		}
		if result.Metrics.MS_SSIM == 0 {
			t.Errorf("No MS-SSIM metric detected: %#v", result)
		}
	})
}

func TestFfmpegVMAF_Negative(t *testing.T) {
	ffmpegExePath, _ := tools.FfmpegPath()
	libvmafModelPath, _ := tools.FindLibvmafModel()

	// Valid tool fixture.
	getValidTool := func() Measurer {
		srcFile := "../../testdata/video/testsrc01.mp4"
		compressedFile := "../../testdata/video/testsrc01.mp4"
		resultFile := t.TempDir() + "/result.json"
		tool, err := NewFfmpegVMAF(ffmpegExePath, libvmafModelPath, compressedFile, srcFile, resultFile)
		if err != nil {
			t.Errorf("Unexpected error when calling NewFfmpegVMAF(): %v", err)
		}
		return tool
	}
	getInvalidTool := func() Measurer {
		srcFile := "nonexistent-source"
		compressedFile := "non-existent-compressed"
		resultFile := t.TempDir() + "/result.json"
		tool, err := NewFfmpegVMAF(ffmpegExePath, libvmafModelPath, compressedFile, srcFile, resultFile)
		if err != nil {
			t.Errorf("Unexpected error when calling NewFfmpegVMAF(): %v", err)
		}
		return tool
	}

	t.Run("Call GetResult() before Measure() should error", func(t *testing.T) {
		tool := getValidTool()
		_, err := tool.GetResult()
		if err == nil {
			t.Fatal("Expecting error if GetResult() called before Measure()")
		}
		gotErrMsg := err.Error()
		wantErrMsg := "GetResult() depends on Measure() called first"
		if diff := cmp.Diff(wantErrMsg, gotErrMsg); diff != "" {
			t.Errorf("Result() error mismatch (-want +got):\n%s", diff)
		}
	})
	t.Run("Second call to Measure() should error", func(t *testing.T) {
		tool := getValidTool()
		// First call is fine.
		if err := tool.Measure(); err != nil {
			t.Fatalf("Unexpected error from first call to Measure(): %v", err)
		}

		// Second call errors.
		if err := tool.Measure(); err == nil {
			t.Error("Expected error from second call to Measure() but go nil")
		}
	})
	t.Run("Calling Measure() on invalid tool should error", func(t *testing.T) {
		tool := getInvalidTool()
		if err := tool.Measure(); err == nil {
			t.Errorf("Expected error when calling Measure() on invalid tool")
		}
	})
}

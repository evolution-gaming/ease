// Copyright ©2022 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package vqm

import (
	"encoding/json"
	"flag"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/evolution-gaming/ease/internal/it"
	"github.com/evolution-gaming/ease/internal/logging"
	"github.com/evolution-gaming/ease/internal/testutil"
)

// Define flags for `go test`.
//
//   -save-result     to save libvmaf result file. This comes handy when need to
//                    add a new version of libvmaf (see testdata/vqm directory).
//   -debug           enable loglevel debug for package tests.
//
// Example:
//
//	go test -run ^TestFfmpegVMAF ./internal/vqm -save-result
//	go test -v -run ^TestFfmpegVMAF ./internal/vqm -debug

var (
	saveResultFile = flag.Bool("save-result", false, "Save result file")
	flDebug        = flag.Bool("debug", false, "Enable debug behaviour, like logging")
)

func TestMain(m *testing.M) {
	flag.Parse()
	if *flDebug {
		logging.EnableDebugLogger()
	}
	os.Exit(m.Run())
}

func TestFfmpegVMAF(t *testing.T) {
	var tool *FfmpegVMAF // tool under test
	var aggMetrics *AggregateMetric

	ffmpegExePath := testutil.EnsureFfmpegWithVMAF(t)
	wrkDir := t.TempDir()

	srcFile := "../../testdata/video/testsrc01.mp4"
	compressedFile := "../../testdata/video/testsrc01.mp4"
	resultFile := path.Join(wrkDir, "result.json")
	if *saveResultFile {
		cwd, _ := os.Getwd()
		resultFile = path.Join(cwd, "result.json")
	}

	t.Run("NewFfmpegVMAF creates new VQM tool", func(t *testing.T) {
		var err error
		tool, err = NewFfmpegVMAF(&FfmpegVMAFConfig{
			FfmpegPath:         ffmpegExePath,
			FfmpegVMAFTemplate: DefaultFfmpegVMAFTemplate,
			ResultFile:         resultFile,
		}, compressedFile, srcFile)
		it.Should(t, err == nil, "Unexpected error: got = %v, want = nil", err)
	})

	t.Run("Call Measure()", func(t *testing.T) {
		err := tool.Measure()
		it.Should(t, err == nil, "Unexpected error: got = %v, want = nil", err)
	})

	t.Run("Call GetMetrics()", func(t *testing.T) {
		var err error
		aggMetrics, err = tool.GetMetrics()
		it.Should(t, err == nil, "Unexpected error: got = %v, want = nil", err)
	})

	t.Run("Aggregate metrics should be non-zero", func(t *testing.T) {
		it.Must(t, aggMetrics != nil, "aggMetrics should not be nil")
		it.Should(t, aggMetrics.VMAF.Mean != 0, "No VMAF metric detected")
		it.Should(t, aggMetrics.PSNR.Mean != 0, "No PSNR metric detected")
	})
}

func TestFfmpegVMAF_WithMSSSIM(t *testing.T) {
	ffmpegExePath := testutil.EnsureFfmpegWithVMAF(t)
	srcFile := "../../testdata/video/testsrc01.mp4"
	compressedFile := "../../testdata/video/testsrc01.mp4"

	// Enable MS-SSIM calculation feature, which is not enabled by default.
	ffmpegVMAFTemplate := "-hide_banner -i {{.CompressedFile}} -i {{.SourceFile}} " +
		"-lavfi libvmaf=n_subsample=1:log_path={{.ResultFile}}:feature=name=psnr|name=float_ms_ssim:" +
		"log_fmt=json:n_threads={{.NThreads}} -f null -"

	tool, err := NewFfmpegVMAF(&FfmpegVMAFConfig{
		FfmpegPath:         ffmpegExePath,
		FfmpegVMAFTemplate: ffmpegVMAFTemplate,
		ResultFile:         path.Join(t.TempDir(), "result_2.json"),
	}, compressedFile, srcFile)
	it.Should(t, err == nil, "Unexpected error: got = %v, want = nil", err)

	err = tool.Measure()
	it.Should(t, err == nil, "Unexpected error from first call to Measure(): got = %v, want = nil", err)

	aggMetrics, err := tool.GetMetrics()
	it.Should(t, err == nil, "Unexpected error: got = %v, want = nil", err)
	it.Should(t, aggMetrics.VMAF.Mean != 0, "No VMAF metric detected")
	it.Should(t, aggMetrics.PSNR.Mean != 0, "No PSNR metric detected")
	it.Should(t, aggMetrics.MS_SSIM.Mean != 0, "No MS-SSIM metric detected")
}

func TestFfmpegVMAF_Negative(t *testing.T) {
	ffmpegExePath := testutil.EnsureFfmpegWithVMAF(t)

	// Valid tool fixture.
	getValidTool := func() *FfmpegVMAF {
		srcFile := "../../testdata/video/testsrc01.mp4"
		compressedFile := "../../testdata/video/testsrc01.mp4"
		resultFile := t.TempDir() + "/result.json"
		tool, err := NewFfmpegVMAF(&FfmpegVMAFConfig{
			FfmpegPath:         ffmpegExePath,
			FfmpegVMAFTemplate: DefaultFfmpegVMAFTemplate,
			ResultFile:         resultFile,
		}, compressedFile, srcFile)

		it.Should(t, err == nil, "Unexpected error: got = %v, want = nil", err)

		return tool
	}

	// Invalid tool fixture.
	getInvalidTool := func() *FfmpegVMAF {
		srcFile := "nonexistent-source"
		compressedFile := "non-existent-compressed"
		resultFile := t.TempDir() + "/result.json"
		tool, err := NewFfmpegVMAF(&FfmpegVMAFConfig{
			FfmpegPath:         ffmpegExePath,
			FfmpegVMAFTemplate: DefaultFfmpegVMAFTemplate,
			ResultFile:         resultFile,
		}, compressedFile, srcFile)

		it.Should(t, err == nil, "Unexpected error: got = %v, want = nil", err)

		return tool
	}

	t.Run("Call GetMetrics() before Measure() should error", func(t *testing.T) {
		wantErrMsg := "GetMetrics() depends on Measure() called first"
		tool := getValidTool()
		_, err := tool.GetMetrics()
		it.Must(t, err != nil, "Should have error")
		it.Should(t, strings.Contains(err.Error(), wantErrMsg), "got = %v, want to contain %v", err, wantErrMsg)
	})
	t.Run("Second call to Measure() should error", func(t *testing.T) {
		tool := getValidTool()
		// First call is fine.
		err := tool.Measure()
		it.Should(t, err == nil, "Unexpected error from first call to Measure(): got = %v, want = nil", err)

		// Second call errors.
		err = tool.Measure()
		it.Should(t, err != nil, "Expected error from second call to Measure()")
	})
	t.Run("Calling Measure() on invalid tool should error", func(t *testing.T) {
		tool := getInvalidTool()
		err := tool.Measure()
		it.Should(t, err != nil, "Expected error calling Measure() on invalid tool")
	})
}

// Different libvmaf versions will generate slightly different outputs. Have to support
// and test accordingly.
func Test_ffmpegVMAFResult_UnmarshalVersions(t *testing.T) {
	tests := map[string]struct {
		resultFile string
	}{
		"2.3.0": {
			resultFile: "../../testdata/vqm/libvmaf_v2.3.0.json",
		},
		"2.3.1": {
			resultFile: "../../testdata/vqm/libvmaf_v2.3.1.json",
		},
		"3.0.0": {
			resultFile: "../../testdata/vqm/libvmaf_v3.0.0.json",
		},
		"3.2.0": {
			resultFile: "../../testdata/vqm/libvmaf_v3.2.0.json",
		},
	}

	for version, tt := range tests {
		t.Run(version, func(t *testing.T) {
			jsonDoc, err := os.ReadFile(tt.resultFile)
			it.Should(t, err == nil, "Unexpected error: got = %v, want = nil", err)

			res := &ffmpegVMAFResult{}
			err2 := json.Unmarshal(jsonDoc, res)
			it.Must(t, err2 == nil, "Unexpected error: got = %v, want = nil", err2)

			it.Should(t, res.Version == version, "got = %v, want = %v", res.Version, version)

			// Check that per-frame VQM values were properly unmarshalled
			// (should not be 0). Only VMAF and PSNR are verified, since MS-SSIM
			// measurement is disabled in default configuration.
			for _, v := range res.Frames {
				it.Should(t, v.Metrics.VMAF != 0, "VMAF should not be zero")
				it.Should(t, v.Metrics.PSNR != 0, "PSNR should not be zero")
			}

			// Check that pooled metric values were properly unmarshalled (should not be 0).
			it.Should(t, res.PooledMetrics.VMAF.Min != 0, "PooledMetrics.VMAF.Min should not be zero")
			it.Should(t, res.PooledMetrics.VMAF.Max != 0, "PooledMetrics.VMAF.Max should not be zero")
			it.Should(t, res.PooledMetrics.VMAF.Mean != 0, "PooledMetrics.VMAF.Mean should not be zero")
			it.Should(t, res.PooledMetrics.VMAF.HarmonicMean != 0, "PooledMetrics.VMAF.HarmonicMean should not be zero")

			it.Should(t, res.PooledMetrics.PSNR.Min != 0, "PooledMetrics.PSNR.Min should not be zero")
			it.Should(t, res.PooledMetrics.PSNR.Max != 0, "PooledMetrics.PSNR.Max should not be zero")
			it.Should(t, res.PooledMetrics.PSNR.Mean != 0, "PooledMetrics.PSNR.Mean should not be zero")
			it.Should(t, res.PooledMetrics.PSNR.HarmonicMean != 0, "PooledMetrics.PSNR.HarmonicMean should not be zero")
		})
	}
}

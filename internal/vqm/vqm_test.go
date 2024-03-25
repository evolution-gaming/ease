// Copyright ©2022 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package vqm

import (
	"encoding/json"
	"flag"
	"os"
	"path"
	"testing"

	"github.com/evolution-gaming/ease/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Define flag for `go test` to save libvmaf result file. This comes handy when need to
// add a new version of libvmaf (see testdata/vqm directory).
//
// Example:
//
//	go test -run ^TestFfmpegVMAF ./internal/vqm -save-result
var saveResultFile = flag.Bool("save-result", false, "Save result file")

func TestFfmpegVMAF(t *testing.T) {
	var tool *FfmpegVMAF // tool under test
	var aggMetrics *AggregateMetric

	wrkDir := t.TempDir()
	ffmpegExePath, _ := tools.FfmpegPath()
	libvmafModelPath, _ := tools.FindLibvmafModel()

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
			LibvmafModelPath:   libvmafModelPath,
			FfmpegVMAFTemplate: DefaultFfmpegVMAFTemplate,
			ResultFile:         resultFile,
		}, compressedFile, srcFile)
		assert.NoError(t, err)
	})

	t.Run("Call Measure()", func(t *testing.T) {
		err := tool.Measure()
		assert.NoError(t, err)
	})

	t.Run("Call GetMetrics()", func(t *testing.T) {
		var err error
		aggMetrics, err = tool.GetMetrics()
		assert.NoError(t, err)
	})

	t.Run("Aggregate metrics should be non-zero", func(t *testing.T) {
		assert.NotEqual(t, aggMetrics.VMAF.Mean, float64(0), "No VMAF metric detected")
		assert.NotEqual(t, aggMetrics.PSNR.Mean, float64(0), "No PSNR metric detected")
	})
}

func TestFfmpegVMAF_WithMSSSIM(t *testing.T) {
	ffmpegExePath, _ := tools.FfmpegPath()
	libvmafModelPath, _ := tools.FindLibvmafModel()
	srcFile := "../../testdata/video/testsrc01.mp4"
	compressedFile := "../../testdata/video/testsrc01.mp4"

	// Enable MS-SSIM calculation feature, which is not enabled by default.
	ffmpegVMAFTemplate := "-hide_banner -i {{.CompressedFile}} -i {{.SourceFile}} " +
		"-lavfi libvmaf=n_subsample=1:log_path={{.ResultFile}}:feature=name=psnr|name=float_ms_ssim:" +
		"log_fmt=json:model=path={{.ModelPath}}:n_threads={{.NThreads}} -f null -"

	tool, err := NewFfmpegVMAF(&FfmpegVMAFConfig{
		FfmpegPath:         ffmpegExePath,
		LibvmafModelPath:   libvmafModelPath,
		FfmpegVMAFTemplate: ffmpegVMAFTemplate,
		ResultFile:         path.Join(t.TempDir(), "result_2.json"),
	}, compressedFile, srcFile)
	assert.NoError(t, err)

	assert.NoError(t, tool.Measure())

	aggMetrics, err := tool.GetMetrics()
	assert.NoError(t, err)
	assert.NotEqual(t, aggMetrics.VMAF.Mean, float64(0), "No VMAF metric detected")
	assert.NotEqual(t, aggMetrics.PSNR.Mean, float64(0), "No PSNR metric detected")
	assert.NotEqual(t, aggMetrics.MS_SSIM.Mean, float64(0), "No MS-SSIM metric detected")
}

func TestFfmpegVMAF_Negative(t *testing.T) {
	ffmpegExePath, _ := tools.FfmpegPath()
	libvmafModelPath, _ := tools.FindLibvmafModel()

	// Valid tool fixture.
	getValidTool := func() *FfmpegVMAF {
		srcFile := "../../testdata/video/testsrc01.mp4"
		compressedFile := "../../testdata/video/testsrc01.mp4"
		resultFile := t.TempDir() + "/result.json"
		tool, err := NewFfmpegVMAF(&FfmpegVMAFConfig{
			FfmpegPath:         ffmpegExePath,
			LibvmafModelPath:   libvmafModelPath,
			FfmpegVMAFTemplate: DefaultFfmpegVMAFTemplate,
			ResultFile:         resultFile,
		}, compressedFile, srcFile)

		assert.NoError(t, err)

		return tool
	}

	// Invalid tool fixture.
	getInvalidTool := func() *FfmpegVMAF {
		srcFile := "nonexistent-source"
		compressedFile := "non-existent-compressed"
		resultFile := t.TempDir() + "/result.json"
		tool, err := NewFfmpegVMAF(&FfmpegVMAFConfig{
			FfmpegPath:         ffmpegExePath,
			LibvmafModelPath:   libvmafModelPath,
			FfmpegVMAFTemplate: DefaultFfmpegVMAFTemplate,
			ResultFile:         resultFile,
		}, compressedFile, srcFile)

		assert.NoError(t, err)

		return tool
	}

	t.Run("Call GetMetrics() before Measure() should error", func(t *testing.T) {
		wantErrMsg := "GetMetrics() depends on Measure() called first"
		tool := getValidTool()
		_, err := tool.GetMetrics()
		require.Error(t, err)
		assert.ErrorContains(t, err, wantErrMsg)
	})
	t.Run("Second call to Measure() should error", func(t *testing.T) {
		tool := getValidTool()
		// First call is fine.
		assert.NoError(t, tool.Measure(), "Unexpected error from first call to Measure()")

		// Second call errors.
		assert.Error(t, tool.Measure(), "Expected error from second call to Measure()")
	})
	t.Run("Calling Measure() on invalid tool should error", func(t *testing.T) {
		tool := getInvalidTool()
		assert.Error(t, tool.Measure(), "Expected error calling Measure() on invalid tool")
	})
}

// Different libvmaf versions will generate slightly different outputs. Have to support
// and test accordingly.
func Test_ffmpegVMAFResult_UnmarshalVersions(t *testing.T) {
	tests := map[string]struct {
		resultFile string
	}{
		"libvmaf v2.3.0": {
			resultFile: "../../testdata/vqm/libvmaf_v2.3.0.json",
		},
		"libvmaf v2.3.1": {
			resultFile: "../../testdata/vqm/libvmaf_v2.3.1.json",
		},
		"libvmaf v3.0.0": {
			resultFile: "../../testdata/vqm/libvmaf_v3.0.0.json",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			jsonDoc, err := os.ReadFile(tt.resultFile)
			assert.NoError(t, err)

			res := &ffmpegVMAFResult{}
			err2 := json.Unmarshal(jsonDoc, res)
			require.NoError(t, err2)

			// Check that per-frame VQM values were properly unmarshalled (should not be 0).
			for _, v := range res.Frames {
				assert.NotEqual(t, v.Metrics.VMAF, 0)
				assert.NotEqual(t, v.Metrics.PSNR, 0)
				assert.NotEqual(t, v.Metrics.MS_SSIM, 0)
			}

			// Check that pooled metric values were properly unmarshalled (should not be 0).
			assert.NotEqual(t, res.PooledMetrics.MS_SSIM.Min, 0)
			assert.NotEqual(t, res.PooledMetrics.MS_SSIM.Max, 0)
			assert.NotEqual(t, res.PooledMetrics.MS_SSIM.Mean, 0)
			assert.NotEqual(t, res.PooledMetrics.MS_SSIM.HarmonicMean, 0)

			assert.NotEqual(t, res.PooledMetrics.VMAF.Min, 0)
			assert.NotEqual(t, res.PooledMetrics.VMAF.Max, 0)
			assert.NotEqual(t, res.PooledMetrics.VMAF.Mean, 0)
			assert.NotEqual(t, res.PooledMetrics.VMAF.HarmonicMean, 0)

			assert.NotEqual(t, res.PooledMetrics.PSNR.Min, 0)
			assert.NotEqual(t, res.PooledMetrics.PSNR.Max, 0)
			assert.NotEqual(t, res.PooledMetrics.PSNR.Mean, 0)
			assert.NotEqual(t, res.PooledMetrics.PSNR.HarmonicMean, 0)
		})
	}
}

// Copyright ©2022 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package vqm

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	metricsFile = "../../testdata/vqm/libvmaf_v2.3.1.json"
	// Expected count of metrics from metricsFile.
	wantMetricCount = 10
)

func fixLoadVmafJSONMetrics(t *testing.T) io.Reader {
	given, err := os.ReadFile(metricsFile)
	assert.NoError(t, err)
	return bytes.NewReader(given)
}

func TestFrameMetrics_FromFfmpegVMAF(t *testing.T) {
	var got FrameMetrics

	err := got.FromFfmpegVMAF(fixLoadVmafJSONMetrics(t))
	require.NoError(t, err)

	t.Run("Should have correct metrics count", func(t *testing.T) {
		assert.Len(t, got, wantMetricCount)
	})

	t.Run("FrameMetric should have correct fields", func(t *testing.T) {
		for i, v := range got {
			assert.EqualValues(t, i, v.FrameNum)
			assert.Greater(t, v.VMAF, float64(0), "VMAF should be positive")
			assert.Greater(t, v.PSNR, float64(0), "PSNR should be positive")
			assert.Greater(t, v.MS_SSIM, float64(0), "MS-SSIM should be positive")
		}
	})
}

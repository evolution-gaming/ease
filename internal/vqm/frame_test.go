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
	wantMetricCount       = 10
	frameMetricsFile      = "../../testdata/vqm/frame_metrics.json"
	wantFrameMetricsCount = 2158
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

func TestFrameMetrics_ToJSON(t *testing.T) {
	// Check To/FromJSON round trip.
	var (
		gotJSON bytes.Buffer
		metrics FrameMetrics
	)

	t.Run("Marshal-unmarshal roundtrip should work", func(t *testing.T) {
		wantJSON, err := os.ReadFile(frameMetricsFile)
		require.NoError(t, err)

		// Unmarshal into FrameMetrics: JSON -> FrameMetrics.
		err2 := metrics.FromJSON(bytes.NewBuffer(wantJSON))
		require.NoError(t, err2)
		assert.Len(t, metrics, wantFrameMetricsCount, "FrameMetrics count mismatch")

		// Marshal back to JSON: FrameMetrics -> JSON.
		err3 := metrics.ToJSON(&gotJSON)
		require.NoError(t, err3)

		assert.JSONEq(t, string(wantJSON), gotJSON.String())
	})
}

func TestFrameMetrics_FromJSON(t *testing.T) {
	t.Run("Should Unmarshal from valid JSON into FrameMetrics", func(t *testing.T) {
		var fm FrameMetrics
		j, err := os.Open(frameMetricsFile)
		require.NoError(t, err)

		err2 := fm.FromJSON(j)
		require.NoError(t, err2)

		assert.Len(t, fm, wantFrameMetricsCount)
	})
	t.Run("Unsuccessful Unmarshal from empty", func(t *testing.T) {
		var fm FrameMetrics

		j := bytes.NewBuffer([]byte{})
		err := fm.FromJSON(j)
		assert.Error(t, err)
	})
}

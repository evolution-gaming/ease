// Copyright ©2022 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Tests for plotting related functionality.

package analysis

import (
	"encoding/json"
	"os"
	"path"
	"testing"

	"github.com/evolution-gaming/ease/internal/tools"
	"github.com/evolution-gaming/ease/internal/vqm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var frameMetricsFile = "../../testdata/vqm/frame_metrics.json"

// getVmafValues fixture provides slice of VMAF metrics.
func getVmafValues(t *testing.T) []float64 {
	var values []float64
	var metrics vqm.FrameMetrics

	j, err := os.Open(frameMetricsFile)
	require.NoError(t, err)

	err2 := metrics.FromJSON(j)
	require.NoError(t, err2, "Error Unmarshaling metrics")

	for _, v := range metrics {
		values = append(values, v.VMAF)
	}

	return values
}

func Test_CreateHistogramPlot(t *testing.T) {
	vmafs := getVmafValues(t)
	title := "Test plot title"

	t.Run("Creating histogram plot should succeed", func(t *testing.T) {
		got, err := CreateHistogramPlot(vmafs, title)
		require.NoError(t, err)
		assert.Equal(t, title, got.X.Label.Text, "Plot title mismatch")
	})
}

func Test_CreateVqmPlot(t *testing.T) {
	vmafs := getVmafValues(t)
	title := "Test plot title"

	t.Run("Creating VQM plot should succeed", func(t *testing.T) {
		got, err := CreateVqmPlot(vmafs, title)
		require.NoError(t, err)
		assert.Equal(t, title, got.Y.Label.Text, "Plot title mismatch")
	})
}

func Test_CreateCDFPlot(t *testing.T) {
	vmafs := getVmafValues(t)
	title := "Test plot title"

	t.Run("Creating CDF plot should succeed", func(t *testing.T) {
		got, err := CreateCDFPlot(vmafs, title)
		require.NoError(t, err)
		assert.Equal(t, title, got.X.Label.Text, "Plot title mismatch")
	})
}

func Test_MultiPlotVqm(t *testing.T) {
	vmafs := getVmafValues(t)
	outDir := t.TempDir()

	t.Run("Creating VQM multi-plot should succeed", func(t *testing.T) {
		outFile := path.Join(outDir, "vqm.png")
		err := MultiPlotVqm(vmafs, "VMAF", "Test plot title", outFile)
		require.NoError(t, err)

		fi, err2 := os.Stat(outFile)
		require.NoError(t, err2)

		// We can't realistically check generated image, instead will do some
		// reasonable check on file properties.
		assert.Greater(t, fi.Size(), int64(10), "Resulting plot file size too small")
	})
}

func Test_CreateBitratePlot(t *testing.T) {
	videoFile := "../../testdata/video/testsrc02.mp4"
	ffprobePath, err := tools.FfprobePath()
	require.NoError(t, err)
	frameStats, err := GetFrameStats(videoFile, ffprobePath)
	require.NoError(t, err)

	t.Run("Creating bitrate plot should succeed", func(t *testing.T) {
		got, err := CreateBitratePlot(frameStats)
		require.NoError(t, err)
		assert.Equal(t, "Kbps", got.Y.Label.Text, "Plot title mismatch")
	})
}

func Test_CreateFrameSizePlot(t *testing.T) {
	videoFile := "../../testdata/video/testsrc02.mp4"
	ffprobePath, err := tools.FfprobePath()
	require.NoError(t, err)
	frameStats, err := GetFrameStats(videoFile, ffprobePath)
	require.NoError(t, err)

	t.Run("Creating frame size plot should succeed", func(t *testing.T) {
		got, err := CreateFrameSizePlot(frameStats)
		require.NoError(t, err)
		assert.Equal(t, "KB", got.Y.Label.Text, "Plot title mismatch")
	})
}

func Test_MultiPlotBitrate(t *testing.T) {
	outDir := t.TempDir()
	videoFile := "../../testdata/video/testsrc02.mp4"
	ffprobePath, err := tools.FfprobePath()
	require.NoError(t, err)

	t.Run("Should create bitrate multi-plot", func(t *testing.T) {
		outFile := path.Join(outDir, "bitrate.png")
		err := MultiPlotBitrate(videoFile, outFile, ffprobePath)
		require.NoError(t, err)

		fi, err2 := os.Stat(outFile)
		require.NoError(t, err2)

		// We can't realistically check generated image, instead will do some
		// reasonable check on file properties.
		assert.Greater(t, fi.Size(), int64(10), "Resulting plot file size too small")
	})
}

func Test_GetFrameStats(t *testing.T) {
	videoFile := "../../testdata/video/testsrc01.mp4"
	// 10 frames in test video
	wantStatCount := 10

	ffprobePath, err := tools.FfprobePath()
	require.NoError(t, err)
	frameStats, err := GetFrameStats(videoFile, ffprobePath)
	require.NoError(t, err)

	t.Run("Should have FrameStat for each frame", func(t *testing.T) {
		assert.Len(t, frameStats, wantStatCount)
	})

	t.Run("Consecutive frames should have different PTS-es", func(t *testing.T) {
		for i := 0; i < len(frameStats)-1; i++ {
			assert.NotEqual(t, frameStats[i].PtsTime, frameStats[i+1].PtsTime, "Consecutive PTS-es equal!")
		}
	})
}

func Test_getDuration(t *testing.T) {
	type testCase struct {
		given []byte
		want  float64
	}

	tests := map[string]testCase{
		"Increasing from ~0": {
			given: []byte(`[
					{ "pts_time": "0.046000", "duration_time": "0.041708", "size": "48929", "flags": "K__" },
					{ "pts_time": "0.087708", "duration_time": "0.041708", "size": "7331", "flags": "___" },
					{ "pts_time": "0.129417", "duration_time": "0.041708", "size": "6968", "flags": "___" }
			]`),
			want: 0.125125,
		},
		"Increasing from >0": {
			given: []byte(`[
					{ "pts_time": "1683156348.790500", "duration_time": "0.041708", "size": "82949", "flags": "K__" },
					{ "pts_time": "1683156348.832208", "duration_time": "0.041708", "size": "1879", "flags": "___" },
					{ "pts_time": "1683156348.873917", "duration_time": "0.041708", "size": "2245", "flags": "___" }
			]`),
			want: 0.125125,
		},
		"Non-monotonic": {
			given: []byte(`[
					{ "pts_time": "1683156348.873917", "duration_time": "0.041708", "size": "2245", "flags": "___" },
					{ "pts_time": "1683156348.790500", "duration_time": "0.041708", "size": "82949", "flags": "K__" },
					{ "pts_time": "1683156348.832208", "duration_time": "0.041708", "size": "1879", "flags": "___" }
			]`),
			want: 0.125125,
		},
		"Zero PTS-es": {
			given: []byte(`[
					{ "pts_time": "0", "duration_time": "0.041708", "size": "2245", "flags": "___" },
					{ "pts_time": "0", "duration_time": "0.041708", "size": "82949", "flags": "K__" },
					{ "pts_time": "0", "duration_time": "0.041708", "size": "1879", "flags": "___" }
			]`),
			want: 0.125124,
		},
		"Incorrect PTS-es": {
			given: []byte(`[
					{ "pts_time": "1.001", "duration_time": "0.04", "size": "2245", "flags": "___" },
					{ "pts_time": "1.002", "duration_time": "0.04", "size": "82949", "flags": "K__" },
					{ "pts_time": "1.003", "duration_time": "0.04", "size": "1879", "flags": "___" }
			]`),
			want: 0.12,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var frames []FrameStat
			assert.NoError(t, json.Unmarshal(tc.given, &frames))

			got := getDuration(frames)
			assert.InDelta(t, tc.want, got, 0.001)
		})
	}
}

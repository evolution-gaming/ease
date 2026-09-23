// Copyright ©2022 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Tests for plotting related functionality.

package analysis

import (
	"encoding/json"
	"math"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/evolution-gaming/ease/internal/it"
	"github.com/evolution-gaming/ease/internal/tools"
	"github.com/evolution-gaming/ease/internal/vqm"
	"gonum.org/v1/plot/plotter"
)

var frameMetricsFile = "../../testdata/vqm/frame_metrics.json"

// getVmafValues fixture provides slice of VMAF metrics.
func getVmafValues(t *testing.T) plotter.XYs {
	fps := 24.0
	var metrics vqm.FrameMetrics

	j, err := os.Open(frameMetricsFile)
	it.Must(t, err == nil, "Unexpected error: got = %v, want = nil", err)
	defer j.Close()

	err2 := json.NewDecoder(j).Decode(&metrics)
	it.Must(t, err2 == nil, "Error Unmarshaling metrics: got = %v, want = nil", err2)

	values := make(plotter.XYs, 0, len(metrics))
	for i, v := range metrics {
		values = append(values, plotter.XY{
			X: float64(i) / fps,
			Y: v.VMAF,
		})
	}

	return values
}

func Test_CreateHistogramPlot(t *testing.T) {
	vmafs := getVmafValues(t)
	title := "Test plot title"

	t.Run("Creating histogram plot should succeed", func(t *testing.T) {
		got, err := CreateHistogramPlot(vmafs, title)
		it.Must(t, err == nil, "Unexpected error: got = %v, want = nil", err)
		it.Should(t, got.X.Label.Text == title, "Plot title mismatch: got = %v, want = %v", got.X.Label.Text, title)
	})
}

func Test_CreateVqmPlot(t *testing.T) {
	vmafs := getVmafValues(t)
	title := "Test plot title"

	t.Run("Creating VQM plot should succeed", func(t *testing.T) {
		got, err := CreateVqmPlot(vmafs, title)
		it.Must(t, err == nil, "Unexpected error: got = %v, want = nil", err)
		it.Should(t, got.Y.Label.Text == title, "Plot title mismatch: got = %v, want = %v", got.Y.Label.Text, title)
	})
}

func Test_CreateCDFPlot(t *testing.T) {
	vmafs := getVmafValues(t)
	vmafYs := make([]float64, vmafs.Len())
	for i := 0; i < vmafs.Len(); i++ {
		_, vmafYs[i] = vmafs.XY(i)
	}
	title := "Test plot title"

	t.Run("Creating CDF plot should succeed", func(t *testing.T) {
		got, err := CreateCDFPlot(vmafYs, title)
		it.Must(t, err == nil, "Unexpected error: got = %v, want = nil", err)
		it.Should(t, got.X.Label.Text == title, "Plot title mismatch: got = %v, want = %v", got.X.Label.Text, title)
	})
}

func Test_MultiPlotVqm(t *testing.T) {
	vmafs := getVmafValues(t)
	outDir := t.TempDir()

	t.Run("Creating VQM multi-plot should succeed", func(t *testing.T) {
		outFile := path.Join(outDir, "vqm.png")
		err := MultiPlotVqm(vmafs, "VMAF", "Test plot title", outFile)
		it.Must(t, err == nil, "Unexpected error: got = %v, want = nil", err)

		fi, err2 := os.Stat(outFile)
		it.Must(t, err2 == nil, "Unexpected error: got = %v, want = nil", err2)

		// We can't realistically check generated image, instead will do some
		// reasonable check on file properties.
		it.Should(t, fi.Size() > 10, "Resulting plot file size too small: got = %v, want > 10", fi.Size())
	})

	t.Run("Non-existent output dir should error", func(t *testing.T) {
		outFile := path.Join(outDir, "no-such-dir", "vqm.png")
		err := MultiPlotVqm(vmafs, "VMAF", "Test plot title", outFile)
		it.Must(t, err != nil, "Should have error")
		wantErrMsg := "creating plot file"
		it.Should(t, strings.Contains(err.Error(), wantErrMsg), "got = %v, want to contain %v", err, wantErrMsg)
	})
}

func Test_CreateBitratePlot(t *testing.T) {
	videoFile := "../../testdata/video/testsrc02.mp4"
	ffprobePath, err := tools.FfprobePath()
	it.Must(t, err == nil, "Unexpected error: got = %v, want = nil", err)
	frameStats, err := GetFrameStats(videoFile, ffprobePath)
	it.Must(t, err == nil, "Unexpected error: got = %v, want = nil", err)

	t.Run("Creating bitrate plot should succeed", func(t *testing.T) {
		got, err := CreateBitratePlot(frameStats)
		it.Must(t, err == nil, "Unexpected error: got = %v, want = nil", err)
		it.Should(t, got.Y.Label.Text == "Kbps", "Plot title mismatch: got = %v, want = %v", got.Y.Label.Text, "Kbps")
	})
}

func Test_CreateFrameSizePlot(t *testing.T) {
	videoFile := "../../testdata/video/testsrc02.mp4"
	ffprobePath, err := tools.FfprobePath()
	it.Must(t, err == nil, "Unexpected error: got = %v, want = nil", err)
	frameStats, err := GetFrameStats(videoFile, ffprobePath)
	it.Must(t, err == nil, "Unexpected error: got = %v, want = nil", err)

	t.Run("Creating frame size plot should succeed", func(t *testing.T) {
		got, err := CreateFrameSizePlot(frameStats)
		it.Must(t, err == nil, "Unexpected error: got = %v, want = nil", err)
		it.Should(t, got.Y.Label.Text == "KB", "Plot title mismatch: got = %v, want = %v", got.Y.Label.Text, "KB")
	})
}

func Test_MultiPlotBitrate(t *testing.T) {
	outDir := t.TempDir()
	videoFile := "../../testdata/video/testsrc02.mp4"
	ffprobePath, err := tools.FfprobePath()
	it.Must(t, err == nil, "Unexpected error: got = %v, want = nil", err)

	t.Run("Should create bitrate multi-plot", func(t *testing.T) {
		outFile := path.Join(outDir, "bitrate.png")
		err := MultiPlotBitrate(videoFile, outFile, ffprobePath)
		it.Must(t, err == nil, "Unexpected error: got = %v, want = nil", err)

		fi, err2 := os.Stat(outFile)
		it.Must(t, err2 == nil, "Unexpected error: got = %v, want = nil", err2)

		// We can't realistically check generated image, instead will do some
		// reasonable check on file properties.
		it.Should(t, fi.Size() > 10, "Resulting plot file size too small: got = %v, want > 10", fi.Size())
	})
}

func Test_GetFrameStats(t *testing.T) {
	videoFile := "../../testdata/video/testsrc01.mp4"
	// 10 frames in test video
	wantStatCount := 10

	ffprobePath, err := tools.FfprobePath()
	it.Must(t, err == nil, "Unexpected error: got = %v, want = nil", err)
	frameStats, err := GetFrameStats(videoFile, ffprobePath)
	it.Must(t, err == nil, "Unexpected error: got = %v, want = nil", err)

	t.Run("Should have FrameStat for each frame", func(t *testing.T) {
		it.Should(t, len(frameStats) == wantStatCount, "got = %v, want = %v", len(frameStats), wantStatCount)
	})

	t.Run("Consecutive frames should have different PTS-es", func(t *testing.T) {
		for i := 0; i < len(frameStats)-1; i++ {
			it.Should(t, frameStats[i].PtsTime != frameStats[i+1].PtsTime, "Consecutive PTS-es equal!")
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
			err := json.Unmarshal(tc.given, &frames)
			it.Should(t, err == nil, "Unexpected error: got = %v, want = nil", err)

			got := getDuration(frames)
			it.Should(t, math.Abs(got-tc.want) <= 0.001, "got = %v, want = %v (delta 0.001)", got, tc.want)
		})
	}
}

func Test_FrameStat_UnmarshalJSON(t *testing.T) {
	type testCase struct {
		given []byte
		want  FrameStat
	}

	tests := map[string]testCase{
		"2 flags keyframe": {
			given: []byte(`{"pts_time": "0", "duration_time": "0.04", "size": "82949", "flags": "K_" }`),
			want: FrameStat{
				KeyFrame:     true,
				DurationTime: 0.04,
				PtsTime:      0,
				Size:         82949,
			},
		},
		"3 flags keyframe": {
			given: []byte(`{"pts_time": "0", "duration_time": "0.04", "size": "82949", "flags": "K__" }`),
			want: FrameStat{
				KeyFrame:     true,
				DurationTime: 0.04,
				PtsTime:      0,
				Size:         82949,
			},
		},
		"3 flags": {
			given: []byte(`{"pts_time": "1", "duration_time": "0.04", "size": "82949", "flags": "___" }`),
			want: FrameStat{
				KeyFrame:     false,
				DurationTime: 0.04,
				PtsTime:      1,
				Size:         82949,
			},
		},
	}

	run := func(t *testing.T, tc testCase) {
		var fs FrameStat
		err := fs.UnmarshalJSON(tc.given)
		it.Should(t, err == nil, "Unexpected error: got = %v, want = nil", err)
		it.Should(t, fs == tc.want, "got = %v, want = %v", fs, tc.want)
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

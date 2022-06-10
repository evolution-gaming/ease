// Copyright ©2022 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Tests for plotting related functionality.

package analysis

import (
	"log"
	"os"
	"path"
	"testing"

	"github.com/evolution-gaming/ease/internal/vqm"

	"github.com/google/go-cmp/cmp"
)

var frameMetricsFile = "../../testdata/vqm/frame_metrics.json"

// getVmafValues fixture provides slice of VMAF metrics.
func getVmafValues() []float64 {
	var values []float64
	var metrics vqm.FrameMetrics

	j, err := os.Open(frameMetricsFile)
	if err != nil {
		log.Panicf("Error opening metrics file: %v", err)
	}

	if err := metrics.FromJSON(j); err != nil {
		log.Panicf("Error Unmarshaling metrics: %v", err)
	}

	for _, v := range metrics {
		values = append(values, v.VMAF)
	}

	return values
}

func Test_CreateHistogramPlot(t *testing.T) {
	vmafs := getVmafValues()
	title := "Test plot title"

	t.Run("Creating historgram plot should succeed", func(t *testing.T) {
		got, err := CreateHistogramPlot(vmafs, title)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if diff := cmp.Diff(title, got.X.Label.Text); diff != "" {
			t.Errorf("Plot title mismatch (-want +got):\n%s", diff)
		}
	})
}

func Test_CreateVqmPlot(t *testing.T) {
	vmafs := getVmafValues()
	title := "Test plot title"

	t.Run("Creating VQM plot should succeed", func(t *testing.T) {
		got, err := CreateVqmPlot(vmafs, title)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if diff := cmp.Diff(title, got.Y.Label.Text); diff != "" {
			t.Errorf("Plot title mismatch (-want +got):\n%s", diff)
		}
	})
}

func Test_CreateCDFPlot(t *testing.T) {
	vmafs := getVmafValues()
	title := "Test plot title"

	t.Run("Creating CDF plot should succeed", func(t *testing.T) {
		got, err := CreateCDFPlot(vmafs, title)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if diff := cmp.Diff(title, got.X.Label.Text); diff != "" {
			t.Errorf("Plot title mismatch (-want +got):\n%s", diff)
		}
	})
}

func Test_MultiPlotVqm(t *testing.T) {
	vmafs := getVmafValues()
	outDir := t.TempDir()

	t.Run("Creating VQM multi-plot should succeed", func(t *testing.T) {
		outFile := path.Join(outDir, "vqm.png")
		err := MultiPlotVqm(vmafs, "VMAF", "Test plot title", outFile)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		fi, err := os.Stat(outFile)
		if err != nil {
			t.Fatalf("Unexpected error from os.Stat: %v", err)
		}

		// We can't realistically check generated image, instead will do some
		// reasonable check on file properties.
		if fi.Size() <= 10 {
			t.Errorf("Resulting plot file size too small: %+v", fi)
		}
	})
}

func Test_CreateBitratePlot(t *testing.T) {
	videoFile := "../../testdata/video/testsrc02.mp4"
	frameStats, err := GetFrameStats(videoFile)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	t.Run("Creating bitrate plot should succeed", func(t *testing.T) {
		got, err := CreateBitratePlot(frameStats)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if diff := cmp.Diff("Kbps", got.Y.Label.Text); diff != "" {
			t.Errorf("Plot title mismatch (-want +got):\n%s", diff)
		}
	})
}

func Test_CreateFrameSizePlot(t *testing.T) {
	videoFile := "../../testdata/video/testsrc02.mp4"
	frameStats, err := GetFrameStats(videoFile)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	t.Run("Creating frame size plot should succeed", func(t *testing.T) {
		got, err := CreateFrameSizePlot(frameStats)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if diff := cmp.Diff("KB", got.Y.Label.Text); diff != "" {
			t.Errorf("Plot title mismatch (-want +got):\n%s", diff)
		}
	})
}

func Test_MultiPlotBitrate(t *testing.T) {
	outDir := t.TempDir()
	videoFile := "../../testdata/video/testsrc02.mp4"

	t.Run("Should create bitrate multi-plot", func(t *testing.T) {
		outFile := path.Join(outDir, "bitrate.png")
		err := MultiPlotBitrate(videoFile, outFile)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		fi, err := os.Stat(outFile)
		if err != nil {
			t.Fatalf("Unexpected error from os.Stat: %v", err)
		}

		// We can't realistically check generated image, instead will do some
		// reasonable check on file properties.
		if fi.Size() <= 10 {
			t.Errorf("Resulting plot file size too small: %+v", fi)
		}
	})
}

func Test_GetFrameStats(t *testing.T) {
	videoFile := "../../testdata/video/testsrc01.mp4"
	// 10 frames in test video
	wantStatCount := 10
	frameStats, err := GetFrameStats(videoFile)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	t.Run("Should have FrameStat for each frame", func(t *testing.T) {
		if diff := cmp.Diff(wantStatCount, len(frameStats)); diff != "" {
			t.Errorf("frameStats size mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("Consecutive frames should have different PTS-es", func(t *testing.T) {
		for i := 0; i < len(frameStats)-1; i++ {
			if frameStats[i].PtsTime == frameStats[i+1].PtsTime {
				t.Errorf("Consecutive PTS-es equal!\npts1: %v\npts2: %v", frameStats[i], frameStats[i+1])
			}
		}
	})
}

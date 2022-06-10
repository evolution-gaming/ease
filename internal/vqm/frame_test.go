// Copyright ©2022 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package vqm

import (
	"bytes"
	"io"
	"log"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var (
	metricsFile = "../../testdata/vqm/ffmpeg_vmaf.json"
	// Expected count of metrics from metricsFile.
	wantMetricCount       = 10
	frameMetricsFile      = "../../testdata/vqm/frame_metrics.json"
	wantFrameMetricsCount = 2158
)

func fixLoadVmafJSONMetrics() io.Reader {
	given, err := os.ReadFile(metricsFile)
	if err != nil {
		log.Panicf("Error reading %s: %s", metricsFile, err)
	}
	return bytes.NewReader(given)
}

func TestFrameMetrics_FromFfmpegVMAF(t *testing.T) {
	var got FrameMetrics

	if err := got.FromFfmpegVMAF(fixLoadVmafJSONMetrics()); err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}

	t.Run("Should have correct metrics count", func(t *testing.T) {
		gotMetricCount := len(got)
		if diff := cmp.Diff(wantMetricCount, gotMetricCount); diff != "" {
			t.Errorf("FrameMetric count mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("FrameMetric should have correct fields", func(t *testing.T) {
		for i, v := range got {
			if diff := cmp.Diff(uint(i), v.FrameNum); diff != "" {
				t.Errorf("FrameNum mismatch (-want +got):\n%s", diff)
			}
			if !(v.VMAF > 0) {
				t.Errorf("VMAF should be positive, got: %v", v.VMAF)
			}
			if !(v.PSNR > 0) {
				t.Errorf("PSNR should be positive, got: %v", v.VMAF)
			}
			if !(v.MS_SSIM > 0) {
				t.Errorf("MS_SSIM should be positive, got: %v", v.VMAF)
			}
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
		if err != nil {
			t.Fatal(err)
		}

		// Unmarshal into FrameMetrics: JSON -> FrameMetrics.
		if err := metrics.FromJSON(bytes.NewBuffer(wantJSON)); err != nil {
			t.Fatalf("Unable to unmarshal: %v", err)
		}

		if diff := cmp.Diff(wantFrameMetricsCount, len(metrics)); diff != "" {
			t.Errorf("FrameMetrics count mismatch (-want +got):\n%s", diff)
		}
		// Marshal back to JSON: FrameMetrics -> JSON.
		if err := metrics.ToJSON(&gotJSON); err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(wantJSON, gotJSON.Bytes()); diff != "" {
			t.Errorf("FrameMetrics JSON mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestFrameMetrics_FromJSON(t *testing.T) {
	t.Run("Should Unmarshal from valid JSON into FrameMetrics", func(t *testing.T) {
		var fm FrameMetrics
		j, err := os.Open(frameMetricsFile)
		if err != nil {
			t.Fatalf("Unexpected error opening JSON file: %v", err)
		}

		if err := fm.FromJSON(j); err != nil {
			t.Fatalf("Unexpected error unmarshaling JSON: %v", err)
		}

		wantCount := wantFrameMetricsCount
		gotCount := len(fm)

		if diff := cmp.Diff(wantCount, gotCount); diff != "" {
			t.Errorf("Metrics count mismatch (-want +got):\n%s", diff)
		}
	})
	t.Run("Unsuccessful Unmarshal from empty", func(t *testing.T) {
		var fm FrameMetrics

		j := bytes.NewBuffer([]byte{})
		err := fm.FromJSON(j)
		if err == nil {
			t.Error("Expecting error, got <nil>")
		}
	})
}

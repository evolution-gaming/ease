// Copyright ©2022 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package vqm

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/evolution-gaming/ease/internal/it"
)

var (
	metricsFile = "../../testdata/vqm/libvmaf_v2.3.1.json"
	// Expected count of metrics from metricsFile.
	wantMetricCount = 10
)

func fixLoadVmafJSONMetrics(t *testing.T) io.Reader {
	given, err := os.ReadFile(metricsFile)
	it.Should(t, err == nil, "Unexpected error: got = %v, want = nil", err)
	return bytes.NewReader(given)
}

func TestFrameMetrics_FromFfmpegVMAF(t *testing.T) {
	var got FrameMetrics

	err := got.FromFfmpegVMAF(fixLoadVmafJSONMetrics(t))
	it.Must(t, err == nil, "Unexpected error: got = %v, want = nil", err)

	t.Run("Should have correct metrics count", func(t *testing.T) {
		it.Should(t, len(got) == wantMetricCount, "got = %v, want = %v", len(got), wantMetricCount)
	})

	t.Run("FrameMetric should have correct fields", func(t *testing.T) {
		for i, v := range got {
			it.Should(t, uint(i) == v.FrameNum, "got = %v, want = %v", v.FrameNum, i)
			it.Should(t, v.VMAF > 0, "VMAF should be positive: got = %v", v.VMAF)
			it.Should(t, v.PSNR > 0, "PSNR should be positive: got = %v", v.PSNR)
			it.Should(t, v.MS_SSIM > 0, "MS-SSIM should be positive: got = %v", v.MS_SSIM)
		}
	})
}

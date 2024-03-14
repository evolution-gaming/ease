// Copyright ©2022 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Video frame related abstractions.

package vqm

import (
	"encoding/json"
	"fmt"
	"io"
)

// FrameMetric contains VQMs for a single frame.
type FrameMetric struct {
	FrameNum uint
	VMAF     float64
	PSNR     float64
	MS_SSIM  float64
}

type FrameMetrics []FrameMetric

// FromFfmpegVMAF will Unmarshal libvmaf's JSON into FrameMetrics.
func (fm *FrameMetrics) FromFfmpegVMAF(jsonReader io.Reader) error {
	b, err := io.ReadAll(jsonReader)
	if err != nil {
		return fmt.Errorf("FromFfmpegVMAF() reading: %w", err)
	}
	res := &ffmpegVMAFResult{}

	if err := json.Unmarshal(b, res); err != nil {
		return fmt.Errorf("FromFfmpegVMAF() unmarshal JSON: %w", err)
	}

	for _, v := range res.Frames {
		*fm = append(*fm, FrameMetric{
			FrameNum: v.FrameNum,
			VMAF:     v.Metrics.VMAF,
			PSNR:     v.Metrics.PSNR,
			MS_SSIM:  v.Metrics.MS_SSIM,
		})
	}
	return nil
}

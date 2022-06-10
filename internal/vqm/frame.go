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

func (fm *FrameMetrics) FromJSON(r io.Reader) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("FromJSON() Read from io.Reader: %w", err)
	}

	if err := json.Unmarshal(data, fm); err != nil {
		return fmt.Errorf("FromJSON() JSON unmarshal: %w", err)
	}

	return nil
}

func (fm *FrameMetrics) Get() []FrameMetric {
	return []FrameMetric(*fm)
}

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

func (fm *FrameMetrics) ToJSON(w io.Writer) error {
	jDoc, err := json.MarshalIndent(fm, "", "  ")
	if err != nil {
		return fmt.Errorf("ToJSON() marshal: %w", err)
	}

	if _, err := w.Write(jDoc); err != nil {
		return fmt.Errorf("ToJSON() write to Writer: %w", err)
	}

	return nil
}

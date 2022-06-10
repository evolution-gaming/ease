// Copyright ©2022 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Video metadata related constructs.

package video

// Metadata type contains useful video stream metadata.
type Metadata struct {
	CodecName string  `json:"codec_name,omitempty"`
	FrameRate string  `json:"r_frame_rate,omitempty"`
	Duration  float64 `json:"duration,omitempty,string"`
	Width     int     `json:"width,omitempty"`
	Height    int     `json:"height,omitempty"`
	BitRate   int     `json:"bit_rate,omitempty,string"`
}

// MetadataExtractor is the interface that wraps ExtractMetadata method.
type MetadataExtractor interface {
	ExtractMetadata(videoFile string) (Metadata, error)
}

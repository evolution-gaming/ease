// Copyright ©2022 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Video metadata related constructs.

package video

// Metadata type contains useful video stream metadata.
type Metadata struct {
	CodecName  string
	FrameRate  string
	Duration   float64
	Width      int
	Height     int
	BitRate    int
	FrameCount int
}

// MetadataExtractor is the interface that wraps ExtractMetadata method.
type MetadataExtractor interface {
	ExtractMetadata(videoFile string) (Metadata, error)
}

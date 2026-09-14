// Copyright ©2026 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Package testutil defines testing helpers.
package testutil

import (
	"fmt"
	"sync"
	"testing"

	"github.com/evolution-gaming/ease/internal/tools"
)

var (
	vmafOnce      sync.Once
	errVmafCheck  error
	ffmpegExePath string
)

// EnsureFfmpegWithVMAF will check if available FFmpeg binary includes libvmaf.
// Any test that requires FFmpeg with VMAF support should call this helper.
func EnsureFfmpegWithVMAF(t *testing.T) string {
	t.Helper()
	// Do the expensive FFmpeg and VMAF check only once, cache reults in package
	// scoped variables.
	vmafOnce.Do(func() {
		fp, err := tools.FfmpegPath()
		if err != nil {
			errVmafCheck = fmt.Errorf("failed to locate FFmpeg binary: %w", err)
			return
		}
		ffmpegExePath = fp

		// Check that VMAF is supported by available FFmpeg binary.
		if err = tools.CheckFfmpegVMAFSupport(ffmpegExePath); err != nil {
			errVmafCheck = fmt.Errorf("VMAF check: %w", err)
		}
	})

	if errVmafCheck != nil {
		t.Fatalf("%v", errVmafCheck)
	}

	return ffmpegExePath
}

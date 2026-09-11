package testutil

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
	"testing"
	"time"

	"github.com/evolution-gaming/ease/internal/tools"
)

var (
	vmafOnce     sync.Once
	vmafCheckErr error
)

// EnsureFFmpegWithVMAF will check if available FFmpeg binary includes libvmaf.
// Any test that requires FFmpeg with VMAF support should call this helper.
func EnsureFFmpegWithVMAF(t *testing.T) {
	t.Helper()
	// Do the expensive FFmpeg and VMAF check only once, cache reults in package
	// scoped variables.
	vmafOnce.Do(func() {
		ffmpegExePath, err := tools.FfmpegPath()
		if err != nil {
			vmafCheckErr = fmt.Errorf("Failed to locale FFmpeg binary: %w", err)
			return
		}

		// Check that VMAF is supported by available FFmpeg binary.
		ctx, cancel := context.WithTimeout(t.Context(), 1*time.Second)
		defer cancel()

		cmd := exec.CommandContext(
			ctx, ffmpegExePath,
			"-f", "lavfi", "-i", "testsrc=size=64x64:rate=1:duration=1",
			"-f", "lavfi", "-i", "testsrc=size=64x64:rate=1:duration=1",
			"-lavfi", "libvmaf", "-f", "null", "-",
		)

		if err = cmd.Run(); err != nil {
			vmafCheckErr = fmt.Errorf("FFmpeg binary (%s) not usable for VMAF calculation: %w", ffmpegExePath, err)
		}
	})

	if vmafCheckErr != nil {
		t.Fatalf("%v", vmafCheckErr)
	}
}

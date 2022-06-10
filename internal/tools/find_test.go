// Copyright ©2022 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package tools

import (
	"os"
	"path"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Test_findTool(t *testing.T) {
	// Create a fake ffprobe binary.
	fakeBinDir := t.TempDir()
	exePath := path.Join(fakeBinDir, "sh")
	f, err := os.OpenFile(exePath, os.O_CREATE, 0o755)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	t.Run("Should fail if executable not found in $PATH nor overridden", func(t *testing.T) {
		got, err := FindTool("nonexistent", "")
		if diff := cmp.Diff("", got); diff != "" {
			t.Errorf("findExecutable() mismatch (-want +got):\n%s", diff)
		}
		if err == nil {
			t.Error("Expecting error")
		}
	})

	t.Run("Should return path if overridden via env var", func(t *testing.T) {
		t.Setenv("CUSTOM_EXE_PATH", exePath)

		got, err := FindTool("sh", "CUSTOM_EXE_PATH")
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(exePath, got); diff != "" {
			t.Errorf("findExecutable() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("Should return path from $PATH", func(t *testing.T) {
		sysPath := os.Getenv("PATH")
		t.Setenv("PATH", fakeBinDir+":"+sysPath)

		got, err := FindTool("sh", "")
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(exePath, got); diff != "" {
			t.Errorf("findExecutable() mismatch (-want +got):\n%s", diff)
		}
	})
}

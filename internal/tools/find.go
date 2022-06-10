// Copyright ©2022 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package tools

import (
	"fmt"
	"os"
	"os/exec"
)

// FindTool will find tool executable in $PATH with possibility to override it
// via environment variable.
func FindTool(exeName, overrideEnvVar string) (string, error) {
	// First check for executable in case it's overridden via env variable.
	if p := os.Getenv(overrideEnvVar); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	// Look for executable in $PATH.
	if p, err := exec.LookPath(exeName); err == nil {
		return p, nil
	}

	// So we did not find any traces of executable - error out!
	return "", fmt.Errorf("binary (%s) not found", exeName)
}

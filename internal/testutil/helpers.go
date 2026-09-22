// Copyright ©2026 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Package testutil also contains some test helpers/predicates as in this file.
package testutil

import "os"

// FileExists is a predicate to check file existence.
func FileExists(fpath string) bool {
	info, err := os.Lstat(fpath)
	if err != nil {
		return false
	}

	if info.IsDir() {
		return false
	}
	return true
}

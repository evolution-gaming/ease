// Copyright ©2026 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Package it contains lean test assertions.
//
// Package API surface is intentionally small, just 2 functions. The aim is to
// avoid polluting with more, predicated can be implemented as see fit per each
// test package or implemneted in some other package.
package it

import (
	"testing"
)

// Should is a soft-assert function.
func Should(t testing.TB, ok bool, format string, args ...any) {
	t.Helper()
	if !ok {
		t.Errorf(format, args...)
	}
}

// Must is a hard-assert function.
func Must(t testing.TB, ok bool, format string, args ...any) {
	t.Helper()
	if !ok {
		t.Fatalf(format, args...)
	}
}

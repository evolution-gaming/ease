// Copyright ©2022 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// A naìve LimitedWriter implementation.
//
// A symmetrical implementation to io.LimitedReader.
package lw

import (
	"errors"
	"io"
)

var ErrLimitedWriterOverflow = errors.New("LimitedWriter overflow")

type LimitedWriter struct {
	// Apply limits to this Writer
	W io.Writer
	// Limit value, does not makes sense to be negative
	N uint
}

// Write implements io.Writer for *LimitedWriter.
func (s *LimitedWriter) Write(b []byte) (int, error) {
	if uint(len(b)) > s.N {
		return 0, ErrLimitedWriterOverflow
	}
	n, err := s.W.Write(b)
	s.N -= uint(n)
	return n, err
}

func LimitWriter(w io.Writer, n uint) io.Writer {
	return &LimitedWriter{w, n}
}

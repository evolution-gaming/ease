// Copyright ©2022 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package lw_test

import (
	"bytes"
	"errors"
	"io"
	"testing"
	"testing/quick"

	"github.com/evolution-gaming/ease/internal/lw"
)

func TestLimitedWriterImplementsWriter(t *testing.T) {
	var _ io.Writer = &lw.LimitedWriter{}
}

func TestLimitedWriterProp(t *testing.T) {
	// How many iterations quick.Check should run.
	iterations := 1 * 1000
	qCfg := &quick.Config{MaxCount: iterations}

	writerFixture := func(size uint) (io.Writer, *bytes.Buffer) {
		buf := &bytes.Buffer{}
		return lw.LimitWriter(buf, size), buf
	}

	t.Run(
		"Written data to large enough buffer should be equal source data",
		func(t *testing.T) {
			fn := func(b []byte) bool {
				// Large enough buffer to hold all data.
				w, buf := writerFixture(uint(len(b)))
				n, err := w.Write(b)
				if err != nil {
					return false
				}
				return n == len(b) && bytes.Equal(b, buf.Bytes())
			}
			if err := quick.Check(fn, qCfg); err != nil {
				t.Error(err)
			}
		})
	t.Run(
		"Multiple writes to large enough buffer",
		func(t *testing.T) {
			fn := func(b []byte, c uint8) bool {
				// Skip empty data.
				if len(b) == 0 {
					return true
				}
				// Large enough buffer to hold all data.
				size := uint(len(b) * int(c))
				w, buf := writerFixture(size)
				for i, fill := c, buf.Len(); i > 0; i-- {
					n, err := w.Write(b)
					if err != nil || n == 0 || !(buf.Len() > fill) {
						return false
					}
				}
				return true
			}
			if err := quick.Check(fn, qCfg); err != nil {
				t.Error(err)
			}
		})

	t.Run(
		"Buffer overflow should return error",
		func(t *testing.T) {
			fn := func(b []byte) bool {
				s := 1
				w, _ := writerFixture(uint(s))
				_, err := w.Write(b)

				if len(b) > s && !errors.Is(err, lw.ErrLimitedWriterOverflow) {
					return false
				}
				return true
			}

			if err := quick.Check(fn, qCfg); err != nil {
				t.Error(err)
			}
		})

	t.Run(
		"Multiple writes with buffer overflow",
		func(t *testing.T) {
			fn := func(b []byte, c uint8) bool {
				// Skip empty data.
				if len(b) == 0 {
					return true
				}
				// Large enough buffer to hold all data.
				size := uint(len(b) * int(c))
				w, _ := writerFixture(size)
				for i := c; i > 0; i-- {
					n, err := w.Write(b)
					if err != nil || n == 0 {
						return false
					}
				}
				// This next write should overflow.
				n, err := w.Write(b)
				if !errors.Is(err, lw.ErrLimitedWriterOverflow) || n != 0 {
					return false
				}
				return true
			}
			if err := quick.Check(fn, qCfg); err != nil {
				t.Error(err)
			}
		})
}

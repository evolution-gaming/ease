// Copyright ©2026 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package metric

import (
	"errors"
	"syscall"
	"testing"
	"time"

	"github.com/evolution-gaming/ease/internal/it"
)

func Test_NewUsageStat(t *testing.T) {
	elapsed := 2 * time.Second
	rusage := &syscall.Rusage{
		Utime:  syscall.Timeval{Sec: 1},
		Stime:  syscall.Timeval{Usec: 500_000},
		Maxrss: 1024,
	}

	got, err := NewUsageStat(elapsed, rusage)
	it.Must(t, err == nil, "Unexpected error: got = %v, want = nil", err)
	it.Should(t, got.Utime == time.Second, "got = %v, want = %v", got.Utime, time.Second)
	it.Should(t, got.Stime == 500*time.Millisecond, "got = %v, want = %v", got.Stime, 500*time.Millisecond)
	it.Should(t, got.Elapsed == elapsed, "got = %v, want = %v", got.Elapsed, elapsed)
	it.Should(t, got.MaxRss == 1024, "got = %v, want = %v", got.MaxRss, 1024)
}

func Test_NewUsageStat_NilRusage(t *testing.T) {
	got, err := NewUsageStat(time.Second, nil)
	it.Should(t, errors.Is(err, ErrNoRusage), "got = %v, want = %v", err, ErrNoRusage)
	it.Should(t, got == UsageStat{}, "got = %v, want = %v", got, UsageStat{})
}

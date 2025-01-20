// Copyright ©2025 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Process usage stats, mostly wrapping relevant fields from syscall.Rusage.
package metric

import (
	"syscall"
	"time"
)

// UsageStat contains process resource usage stats.
type UsageStat struct {
	// Human friendly representations of time duration
	HStime   string
	HUtime   string
	HElapsed string
	// time.Duration is nanoseconds
	Stime   time.Duration
	Utime   time.Duration
	Elapsed time.Duration
	// MaxRss is KB
	MaxRss int64
}

// NewUsageStat will create UsageStat instance.
func NewUsageStat(elapsed time.Duration, rusage *syscall.Rusage) UsageStat {
	return UsageStat{
		Stime:    time.Duration(syscall.TimevalToNsec(rusage.Stime)),
		Utime:    time.Duration(syscall.TimevalToNsec(rusage.Utime)),
		Elapsed:  elapsed,
		HStime:   time.Duration(syscall.TimevalToNsec(rusage.Stime)).String(),
		HUtime:   time.Duration(syscall.TimevalToNsec(rusage.Utime)).String(),
		HElapsed: elapsed.String(),
		MaxRss:   rusage.Maxrss,
	}
}

// CPUPercent calculates CPU usage in percent.
func (s *UsageStat) CPUPercent() float64 {
	return float64(s.Stime+s.Utime) / float64(s.Elapsed) * 100
}

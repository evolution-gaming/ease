// Copyright ©2022 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Reusable parts of ease application and subcommand infrastructure.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/evolution-gaming/ease/internal/encoding"
	"github.com/evolution-gaming/ease/internal/logging"
)

// AppError a custom error returned from CLI application.
//
// AppError is handy error type envisioned to be used in CLI's main.
// ExitCode() should be used as argument for os.Exit().
type AppError struct {
	msg      string
	exitCode int
}

// Error implements error interface for AppError.
func (e *AppError) Error() string {
	return e.msg
}

// ExitCode returns CLI application's exit code.
func (e *AppError) ExitCode() int {
	return e.exitCode
}

// printSubCommandUsage helper to format ad print subcommand's usage.
func printSubCommandUsage(longHelp string, fs *flag.FlagSet) {
	fmt.Fprintf(fs.Output(), "Usage of sub-command %s:\n\n", fs.Name())
	fmt.Fprintf(fs.Output(), "%s\n\n", longHelp)
	fs.PrintDefaults()
}

// unrollResultErrors helper to unroll all errors from RunResults into a string.
func unrollResultErrors(results []encoding.RunResult) string {
	sb := strings.Builder{}
	for i := range results {
		rr := &results[i]
		if len(rr.Errors) != 0 {
			for _, e := range rr.Errors {
				sb.WriteString(fmt.Sprintf("%s:\n\t%s\n", rr.Name, e.Error()))
			}
		}
	}
	return sb.String()
}

// createPlanConfig creates a PlanConfig instance from JSON configuration.
func createPlanConfig(cfgFile string) (pc encoding.PlanConfig, err error) {
	fd, err := os.Open(cfgFile)
	if err != nil {
		return pc, fmt.Errorf("cannot open conf file: %w", err)
	}
	defer fd.Close()

	jdoc, err := io.ReadAll(fd)
	if err != nil {
		return pc, fmt.Errorf("cannot read data from conf file: %w", err)
	}

	pc, err = encoding.NewPlanConfigFromJSON(jdoc)
	if err != nil {
		return pc, fmt.Errorf("cannot create PlanConfig: %w", err)
	}

	if ok, err := pc.IsValid(); !ok {
		ev := &encoding.PlanConfigError{}
		if errors.As(err, &ev) {
			logging.Debugf(
				"PlanConfig validation failures:\n%s",
				strings.Join(ev.Reasons(), "\n"))
		}
		return pc, fmt.Errorf("PlanConfig not valid: %w", err)
	}

	return pc, nil
}

// isNonEmptyDir will check if given directory is non-empty.
func isNonEmptyDir(path string) bool {
	fs, err := os.Open(path)
	if err != nil {
		return false
	}
	defer fs.Close()

	n, _ := fs.Readdirnames(1)
	return len(n) == 1
}

// fileExists is simple helper to check if file exists.
func fileExists(p string) bool {
	info, err := os.Lstat(p)
	if err != nil {
		return false
	}
	if info.IsDir() {
		return false
	}

	return true
}

func all[T comparable](s []T, f func(val T) bool) bool {
	if len(s) == 0 {
		return false
	}

	for _, e := range s {
		if !f(e) {
			return false
		}
	}

	return true
}

// parseFraction will parse fraction string representation.
func parseFraction(x string) (float64, error) {
	numStr, denomStr, found := strings.Cut(x, "/")

	numerator, err := strconv.Atoi(numStr)
	if err != nil {
		return 0, fmt.Errorf("parsing numerator: %w", err)
	}
	// In case it is not a fraction - return numerator.
	if !found {
		return float64(numerator), nil
	}

	denominator, err := strconv.Atoi(denomStr)
	if err != nil {
		return 0, fmt.Errorf("parsing denominator: %w", err)
	}
	if denominator == 0 {
		return 0, fmt.Errorf("zero division for %s", x)
	}

	return float64(numerator) / float64(denominator), nil
}

// Helpers for plotting with gonum, we need to implement plotter.XYer interface.
type (
	metricXYs []metricXY
	metricXY  struct{ X, Y float64 }
)

func (m metricXYs) Len() int {
	return len(m)
}

func (m metricXYs) XY(i int) (float64, float64) {
	return m[i].X, m[i].Y
}

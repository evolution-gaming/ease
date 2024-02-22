// Copyright ©2022 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Reusable parts of ease application and subcommand infrastructure.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/evolution-gaming/ease/internal/encoding"
	"github.com/evolution-gaming/ease/internal/logging"
	"github.com/evolution-gaming/ease/internal/vqm"
	"github.com/jszwec/csvutil"
)

// Commander interface should be implemented by commands and sub-commands.
type Commander interface {
	Run([]string) error
}

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

// namedVqmResult is structure that wraps vqm.Result with a name.
type namedVqmResult struct {
	Name string
	vqm.Result
}

// report contains application execution result.
type report struct {
	EncodingResult encoding.PlanResult
	VQMResults     []namedVqmResult
}

// WriteJSON writes application execution result as JSON.
func (r *report) WriteJSON(w io.Writer) error {
	// Write Plan execution result to JSON (for now)
	res, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling encoding result to JSON: %w", err)
	}
	_, err = w.Write(res)
	if err != nil {
		return fmt.Errorf("writing encoding result %w", err)
	}
	return nil
}

// csvRecord contains result fields from one encode.
type csvRecord struct {
	Name             string
	SourceFile       string
	CompressedFile   string
	Cmd              string
	HStime           string
	HUtime           string
	HElapsed         string
	Stime            time.Duration
	Utime            time.Duration
	Elapsed          time.Duration
	MaxRss           int64
	VideoDuration    float64
	AvgEncodingSpeed float64
	PSNR             float64
	MS_SSIM          float64
	VMAF             float64
}

// Wrap rows of csvRecords mainly to attach relevant methods.
type csvReport struct {
	rows []csvRecord
}

func newCsvReport(r *report) (*csvReport, error) {
	size := len(r.EncodingResult.RunResults)
	if size != len(r.VQMResults) {
		return nil, errors.New("Encoding result and VQM result size do not match")
	}

	var report csvReport
	report.rows = make([]csvRecord, 0, size)

	// Need to create an intermediate mapping from CompressedFile to VQM metrics to make
	// merging fields from two sources easier (we cannot rely on order). CompressedFile
	// being a unique identifier (Name does not work when there are multiple input files).
	tVqms := make(map[string]vqm.VideoQualityMetrics, size)
	for _, v := range r.VQMResults {
		tVqms[v.CompressedFile] = v.Metrics
	}

	// Final loop to merge into a single report.
	for _, v := range r.EncodingResult.RunResults {
		vqm, ok := tVqms[v.CompressedFile]
		if !ok {
			return nil, fmt.Errorf("no VQMs for map key: %s", v.CompressedFile)
		}
		report.rows = append(report.rows, csvRecord{
			Name:             v.Name,
			SourceFile:       v.SourceFile,
			CompressedFile:   v.CompressedFile,
			Cmd:              v.Cmd,
			HStime:           v.Stats.HStime,
			HUtime:           v.Stats.HUtime,
			HElapsed:         v.Stats.HElapsed,
			Stime:            v.Stats.Stime,
			Utime:            v.Stats.Utime,
			Elapsed:          v.Stats.Elapsed,
			MaxRss:           v.Stats.MaxRss,
			VideoDuration:    v.VideoDuration,
			AvgEncodingSpeed: v.AvgEncodingSpeed,
			PSNR:             vqm.PSNR,
			MS_SSIM:          vqm.MS_SSIM,
			VMAF:             vqm.VMAF,
		})
	}

	return &report, nil
}

// WriteCSV saves flat application report representation to io.Writer.
func (r *csvReport) WriteCSV(w io.Writer) error {
	data, err := csvutil.Marshal(r.rows)
	if err != nil {
		return err
	}
	_, err2 := w.Write(data)
	if err2 != nil {
		return err2
	}
	return nil
}

// parseReportFile is a helper to read and parse report JSON file into report type.
func parseReportFile(fPath string) *report {
	var r report

	b, err := os.ReadFile(fPath)
	if err != nil {
		log.Panicf("Unable to read file %s: %v", fPath, err)
	}

	if err := json.Unmarshal(b, &r); err != nil {
		log.Panic(err)
	}

	return &r
}

// sourceData is a helper data structure with fields related to single encoded file.
type sourceData struct {
	CompressedFile string
	WorkDir        string
	VqmResultFile  string
}

// extractSourceData create mapping from compressed file to sourceData.
//
// Since in report file we have separate keys RunResults and VQMResults and we
// need to merge fields from both, we create mapping from unique CompressedFile
// field to sourceData.
func extractSourceData(r *report) map[string]sourceData {
	s := make(map[string]sourceData)
	// Create map to sourceData (incomplete at this point) from RunResults
	for i := range r.EncodingResult.RunResults {
		v := &r.EncodingResult.RunResults[i]
		sd := s[v.CompressedFile]
		sd.WorkDir = v.WorkDir
		sd.CompressedFile = v.CompressedFile
		s[v.CompressedFile] = sd
	}

	// Fill-in missing VqmResultFile field from VQMResult.
	for i := range r.VQMResults {
		v := &r.VQMResults[i]
		sd := s[v.CompressedFile]
		sd.VqmResultFile = v.ResultFile
		s[v.CompressedFile] = sd
	}
	return s
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

func all[T comparable](s []T, val T) bool {
	if len(s) == 0 {
		return false
	}

	for _, e := range s {
		if e != val {
			return false
		}
	}

	return true
}

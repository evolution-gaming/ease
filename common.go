// Copyright ©2022 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Reusable parts of ease application and subcommand infrastructure.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/evolution-gaming/ease/internal/encoding"
	"github.com/evolution-gaming/ease/internal/logging"
	"github.com/evolution-gaming/ease/internal/vqm"
)

// Commander interface should be implemented by commands and sub-commands.
type Commander interface {
	Run([]string) error
	Name() string
	Help()
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
func (r *report) WriteJSON(w io.Writer) {
	// Write Plan execution result to JSON (for now)
	res, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		logging.Infof("Unable to marshal encoding result to JSON: %s", err)
	}
	_, err = w.Write(res)
	if err != nil {
		logging.Infof("Error writing encoding result %s", err)
	}
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

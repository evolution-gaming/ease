// Copyright ©2022 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// ease tool's encode subcommand implementation.

package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/evolution-gaming/ease/internal/encoding"
	"github.com/evolution-gaming/ease/internal/logging"
	"github.com/evolution-gaming/ease/internal/tools"
	"github.com/evolution-gaming/ease/internal/vqm"
)

// CreateEncodeCommand will create Commander instace from EncodeApp.
func CreateEncodeCommand() Commander {
	longHelp := `Subcommand "encode" will execute encoding plan according to definition in file
provided as parameter to -plan flag. This flag is mandatory.

Examples:

  ease encode -plan plan.json -report encode_report.json`
	app := &EncodeApp{
		fs: flag.NewFlagSet("encode", flag.ContinueOnError),
	}
	app.fs.StringVar(&app.flPlan, "plan", "", "Encoding plan configuration file")
	app.fs.StringVar(&app.flReport, "report", "", "Encoding plan report file (default is stdout)")
	app.fs.BoolVar(&app.flCalculateVQM, "vqm", true, "Calculate VQMs")
	app.fs.BoolVar(&app.flDryRun, "dry-run", false, "Do not actually run, just do checks and validation")
	app.fs.Usage = func() {
		printSubCommandUsage(longHelp, app.fs)
	}

	return app
}

// Make sure EncodeApp implements Commander interface.
var _ Commander = (*EncodeApp)(nil)

// EncodeApp is subcommand application context that implements Commander interface.
type EncodeApp struct {
	// FlagSet instance
	fs *flag.FlagSet
	// Encoding plan config file flag
	flPlan string
	// Execution report output file flag
	flReport string
	// Calculate VQM flag
	flCalculateVQM bool
	// Dry run mode flag
	flDryRun bool
}

func (a *EncodeApp) Name() string {
	return a.fs.Name()
}

func (a *EncodeApp) Help() {
	a.fs.Usage()
}

// ReportWriter returns io.Writer for report.
func (a *EncodeApp) ReportWriter() io.Writer {
	var out io.WriteCloser
	// Either write to file or stdout.
	if a.flReport == "" {
		// Case to write to stdout.
		return os.Stdout
	}
	// Case to write to file.
	out, err := os.Create(a.flReport)
	if err != nil {
		logging.Infof("Unable to create result file redirecting to stdout: %s", err)
		return os.Stdout
	}
	return out
}

// Init will do App state initialization.
func (a *EncodeApp) Init(args []string) error {
	if err := a.fs.Parse(args); err != nil {
		return &AppError{
			exitCode: 2,
			msg:      fmt.Sprintf("%s usage error", a.Name()),
		}
	}

	// Encoding plan config file is mandatory.
	if a.flPlan == "" {
		a.Help()
		return &AppError{
			exitCode: 2,
			msg:      "mandatory option -plan is missing",
		}
	}

	// Encoding plan config file should exist.
	if _, err := os.Stat(a.flPlan); err != nil {
		a.Help()
		return &AppError{
			exitCode: 2,
			msg:      fmt.Sprintf("encoding plan file does not exist? %s", err),
		}
	}

	return nil
}

// Run is main entry point into App execution.
func (a *EncodeApp) Run(args []string) error {
	if err := a.Init(args); err != nil {
		return err
	}

	logging.Debugf("Encoding plan config file: %v", a.flPlan)

	plan, err := createPlanFromJSONConfig(a.flPlan)
	if err != nil {
		return &AppError{exitCode: 1, msg: err.Error()}
	}

	// Check external tool dependencies - for VMAF calculations we require
	// ffmpeg and libvmaf model file available.
	ffmpegPath, err := tools.FfmpegPath()
	if err != nil {
		return &AppError{exitCode: 1, msg: fmt.Sprintf("dependency ffmpeg: %s", err)}
	}

	libvmafModelPath, err := tools.FindLibvmafModel()
	if err != nil {
		return &AppError{exitCode: 1, msg: fmt.Sprintf("dependency libvmaf model: %s", err)}
	}

	// Early return in "dry run" mode.
	if a.flDryRun {
		logging.Info("Dry run mode finished!")
		return nil
	}

	result, err := plan.Run()
	// Make sure to log any errors from RunResults.
	if ur := unrollResultErrors(result.RunResults); ur != "" {
		logging.Infof("Run had following ERRORS:\n%s", ur)
	}
	if err != nil {
		return &AppError{exitCode: 1, msg: err.Error()}
	}

	// Do VQM calculations for encoded videos.
	var vqmFailed bool = false
	var vqmResults []namedVqmResult
	if a.flCalculateVQM {
		for i := range result.RunResults {
			r := &result.RunResults[i]
			resFile := strings.TrimSuffix(r.CompressedFile, filepath.Ext(r.CompressedFile)) + "_vqm.json"
			vqmTool, err := vqm.NewFfmpegVMAF(ffmpegPath, libvmafModelPath, r.CompressedFile, r.SourceFile, resFile)
			if err != nil {
				vqmFailed = true
				logging.Infof("Error while initializing VQM tool: %s", err)
				continue
			}

			logging.Infof("Start measuring VQMs for %s", r.CompressedFile)
			if err = vqmTool.Measure(); err != nil {
				vqmFailed = true
				logging.Infof("Failed calculate VQM for %s due to error: %s", r.CompressedFile, err)
				continue
			}

			res, err := vqmTool.GetResult()
			if err != nil {
				logging.Infof("Error while getting VQM result for %s: %s", r.CompressedFile, err)
			}
			vqmResults = append(vqmResults, namedVqmResult{Name: r.Name, Result: res})

			logging.Infof("Done measuring VQMs for %s", r.CompressedFile)
		}
	}
	if vqmFailed {
		return &AppError{
			msg:      "VQM calculations had errors, see log for reasons",
			exitCode: 1,
		}
	}

	// Report encoding application results.
	rep := report{
		EncodingResult: result,
		VQMResults:     vqmResults,
	}
	rep.WriteJSON(a.ReportWriter())

	return nil
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

// createPlanFromJSONConfig creates a Plan instance from JSON configuration.
func createPlanFromJSONConfig(cfgFile string) (encoding.Plan, error) {
	var plan encoding.Plan
	fd, err := os.Open(cfgFile)
	if err != nil {
		return plan, fmt.Errorf("cannot open conf file: %w", err)
	}
	defer fd.Close()

	jdoc, err := io.ReadAll(fd)
	if err != nil {
		return plan, fmt.Errorf("cannot read data from conf file: %w", err)
	}

	pc, err := encoding.NewPlanConfigFromJSON(jdoc)
	if err != nil {
		return plan, fmt.Errorf("cannot create PlanConfig: %w", err)
	}

	if ok, err := pc.IsValid(); !ok {
		ev := &encoding.PlanConfigError{}
		if errors.As(err, &ev) {
			logging.Debugf(
				"PlanConfig validation failures:\n%s",
				strings.Join(ev.Reasons(), "\n"))
		}
		return plan, fmt.Errorf("PlanConfig not valid: %w", err)
	}

	return encoding.NewPlan(pc), nil
}

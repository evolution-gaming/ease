// Copyright ©2022 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// ease tool's run subcommand implementation.

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/evolution-gaming/ease/internal/analysis"
	"github.com/evolution-gaming/ease/internal/encoding"
	"github.com/evolution-gaming/ease/internal/logging"
	"github.com/evolution-gaming/ease/internal/vqm"
)

// CreateRunCommand will create Commander instance from App.
func CreateRunCommand() Commander {
	longHelp := `Subcommand "run" will execute encoding plan according to definition in file
provided as parameter to -plan flag and will calculate and report VQM
metrics. This flag is mandatory.

Examples:

  ease run -plan plan.json -out-dir path/to/output/dir`

	app := &App{
		fs: flag.NewFlagSet("run", flag.ContinueOnError),
		gf: globalFlags{},
	}
	app.gf.Register(app.fs)
	app.fs.StringVar(&app.flPlan, "plan", "", "Encoding plan configuration file")
	app.fs.StringVar(&app.flOutDir, "out-dir", "", "Output directory to store results")
	app.fs.BoolVar(&app.flDryRun, "dry-run", false, "Do not actually run, just do checks and validation")
	app.fs.Usage = func() {
		printSubCommandUsage(longHelp, app.fs)
	}

	return app
}

// Make sure App implements Commander interface.
var _ Commander = (*App)(nil)

// App is subcommand application context that implements Commander interface.
type App struct {
	// Configuration object
	cfg *Config
	// FlagSet instance
	fs *flag.FlagSet
	// Optional configuration file
	flPlan string
	// Output directory for analysis results
	flOutDir string
	// Global flags
	gf globalFlags
	// Dry run mode flag
	flDryRun bool
}

// init will do App state initialization.
func (a *App) init(args []string) error {
	if err := a.fs.Parse(args); err != nil {
		return &AppError{
			exitCode: 2,
			msg:      fmt.Sprintf("%s usage error", a.fs.Name()),
		}
	}

	if a.gf.Debug {
		logging.EnableDebugLogger()
	}

	// Encoding plan config file is mandatory.
	if a.flPlan == "" {
		a.fs.Usage()
		return &AppError{
			exitCode: 2,
			msg:      "mandatory option -plan is missing",
		}
	}

	// Output dir is mandatory.
	if a.flOutDir == "" {
		a.fs.Usage()
		return &AppError{
			exitCode: 2,
			msg:      "mandatory option -out-dir is missing",
		}
	}

	// Encoding plan config file should exist.
	if _, err := os.Stat(a.flPlan); err != nil {
		a.fs.Usage()
		return &AppError{
			exitCode: 2,
			msg:      fmt.Sprintf("encoding plan file does not exist? %s", err),
		}
	}

	// Do not write over existing output directory.
	if isNonEmptyDir(a.flOutDir) {
		return &AppError{exitCode: 1, msg: fmt.Sprintf("non-empty out dir: %s", a.flOutDir)}
	}

	// Load application configuration.
	c, err := LoadConfig(a.gf.ConfFile)
	if err != nil {
		return &AppError{exitCode: 1, msg: err.Error()}
	}
	a.cfg = &c

	return nil
}

// encode will run encoding stage of plan execution.
func (a *App) encode(plan encoding.Plan) (*report, error) {
	rep := &report{}

	result, err := plan.Run()
	// Make sure to log any errors from RunResults.
	if ur := unrollResultErrors(result.RunResults); ur != "" {
		logging.Infof("Run had following ERRORS:\n%s", ur)
	}
	if err != nil {
		return rep, &AppError{exitCode: 1, msg: err.Error()}
	}
	rep.EncodingResult = result

	// Do VQM calculations for encoded videos.
	var vqmFailed bool = false

	for i := range result.RunResults {
		r := &result.RunResults[i]
		resFile := strings.TrimSuffix(r.CompressedFile, filepath.Ext(r.CompressedFile)) + "_vqm.json"

		// Create VMAF tool configuration.
		vmafCfg := vqm.FfmpegVMAFConfig{
			FfmpegPath:         a.cfg.FfmpegPath.Value(),
			LibvmafModelPath:   a.cfg.LibvmafModelPath.Value(),
			FfmpegVMAFTemplate: a.cfg.FfmpegVMAFTemplate.Value(),
			ResultFile:         resFile,
		}

		vqmTool, err2 := vqm.NewFfmpegVMAF(&vmafCfg, r.CompressedFile, r.SourceFile)
		if err2 != nil {
			vqmFailed = true
			logging.Infof("Error while initializing VQM tool: %s", err2)
			continue
		}

		logging.Infof("Start measuring VQMs for %s", r.CompressedFile)
		if err2 = vqmTool.Measure(); err2 != nil {
			vqmFailed = true
			logging.Infof("Failed calculate VQM for %s due to error: %s", r.CompressedFile, err2)
			continue
		}

		res, err2 := vqmTool.GetResult()
		if err2 != nil {
			logging.Infof("Error while getting VQM result for %s: %s", r.CompressedFile, err2)
		}
		rep.VQMResults = append(rep.VQMResults, namedVqmResult{Name: r.Name, Result: res})

		logging.Infof("Done measuring VQMs for %s", r.CompressedFile)
	}

	if vqmFailed {
		return rep, &AppError{
			msg:      "VQM calculations had errors, see log for reasons",
			exitCode: 1,
		}
	}

	// Write report of encoding results.
	reportPath := path.Join(a.flOutDir, a.cfg.ReportFileName.Value())
	reportOut, err := os.Create(reportPath)
	if err != nil {
		return rep, &AppError{
			msg:      fmt.Sprintf("Unable to create report file: %s", err),
			exitCode: 1,
		}
	}
	defer reportOut.Close()
	rep.WriteJSON(reportOut)

	return rep, nil
}

// analyse will run analysis stage of plan execution.
func (a *App) analyse(rep *report) error {
	// Extract data to work with.
	srcData := extractSourceData(rep)
	d, err := json.MarshalIndent(srcData, "", "  ")
	if err != nil {
		return &AppError{
			exitCode: 1,
			msg:      err.Error(),
		}
	}
	logging.Debugf("Analysis for:\n%s", d)

	// TODO: this is a good place to do goroutines iterate over sources and do stuff.

	for _, v := range srcData {
		// Create separate dir for results.
		base := path.Base(v.CompressedFile)
		base = strings.TrimSuffix(base, path.Ext(base))
		logging.Infof("Analysing %s", v.CompressedFile)
		resDir := path.Join(a.flOutDir, base)
		if err := os.MkdirAll(resDir, os.FileMode(0o755)); err != nil {
			return &AppError{
				msg:      fmt.Sprintf("failed creating directory: %s", err),
				exitCode: 1,
			}
		}

		compressedFile := v.CompressedFile
		vqmFile := v.VqmResultFile
		// In case compressed and VQM result file path in not absolute we assume
		// it must be relative to WorkDir.
		if !path.IsAbs(compressedFile) {
			compressedFile = path.Join(v.WorkDir, compressedFile)
		}
		if !path.IsAbs(vqmFile) {
			vqmFile = path.Join(v.WorkDir, vqmFile)
		}
		bitratePlot := path.Join(resDir, base+"_bitrate.png")
		vmafPlot := path.Join(resDir, base+"_vmaf.png")
		psnrPlot := path.Join(resDir, base+"_psnr.png")
		msssimPlot := path.Join(resDir, base+"_ms-ssim.png")

		jsonFd, err := os.Open(vqmFile)
		if err != nil {
			return &AppError{
				msg:      fmt.Sprintf("failed opening VQM file: %s", err),
				exitCode: 1,
			}
		}

		var frameMetrics vqm.FrameMetrics
		err = frameMetrics.FromFfmpegVMAF(jsonFd)
		// Close jsonFd file descriptor at earliest convenience. Should avoid use of defer
		// in loop in this case.
		jsonFd.Close()
		if err != nil {
			return &AppError{
				msg:      fmt.Sprintf("failed converting to FrameMetrics: %s", err),
				exitCode: 1,
			}
		}

		var vmafs, psnrs, msssims []float64
		for _, v := range frameMetrics {
			vmafs = append(vmafs, v.VMAF)
			psnrs = append(psnrs, v.PSNR)
			msssims = append(msssims, v.MS_SSIM)
		}

		if err := analysis.MultiPlotBitrate(compressedFile, bitratePlot, a.cfg.FfprobePath.Value()); err != nil {
			return &AppError{
				msg:      fmt.Sprintf("failed creating bitrate plot: %s", err),
				exitCode: 1,
			}
		}
		logging.Infof("Bitrate plot done: %s", bitratePlot)

		if err := analysis.MultiPlotVqm(vmafs, "VMAF", base, vmafPlot); err != nil {
			return &AppError{
				msg:      fmt.Sprintf("failed creating VMAF multiplot: %s", err),
				exitCode: 1,
			}
		}
		logging.Infof("VMAF multi-plot done: %s", vmafPlot)

		if err := analysis.MultiPlotVqm(psnrs, "PSNR", base, psnrPlot); err != nil {
			return &AppError{
				msg:      fmt.Sprintf("failed creating PSNR multiplot: %s", err),
				exitCode: 1,
			}
		}
		logging.Infof("PSNR multi-plot done: %s", psnrPlot)

		if err := analysis.MultiPlotVqm(msssims, "MS-SSIM", base, msssimPlot); err != nil {
			return &AppError{
				msg:      fmt.Sprintf("failed creating MS-SSIM multiplot: %s", err),
				exitCode: 1,
			}
		}
		logging.Infof("MS-SSIM multi-plot done: %s", msssimPlot)
	}

	return nil
}

// Run is main entry point into App execution.
func (a *App) Run(args []string) error {
	if err := a.init(args); err != nil {
		return err
	}

	logging.Debugf("Application configuration: %#v", a.cfg)
	// Check if configuration is valid.
	if err := a.cfg.Verify(); err != nil {
		return &AppError{exitCode: 1, msg: fmt.Sprintf("configuration validation: %s", err)}
	}

	logging.Debugf("Encoding plan config file: %v", a.flPlan)

	pc, err := createPlanConfig(a.flPlan)
	if err != nil {
		return &AppError{exitCode: 1, msg: err.Error()}
	}

	plan := encoding.NewPlan(pc, a.flOutDir)

	// Early return in "dry run" mode.
	if a.flDryRun {
		logging.Info("Dry run mode finished!")
		return nil
	}

	// Run encode stage.
	rep, err := a.encode(plan)
	if err != nil {
		return &AppError{exitCode: 1, msg: err.Error()}
	}

	// Run analysis stage.
	return a.analyse(rep)
}

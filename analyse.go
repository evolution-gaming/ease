// Copyright ©2022 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// ease tool's analyse subcommand implementation.

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/evolution-gaming/ease/internal/analysis"
	"github.com/evolution-gaming/ease/internal/logging"
	"github.com/evolution-gaming/ease/internal/tools"
	"github.com/evolution-gaming/ease/internal/vqm"
)

// Make sure AnalyseApp implements Commander interface.
var _ Commander = (*AnalyseApp)(nil)

// AnalyseApp is analyse subcommand context that implements Commander interface.
type AnalyseApp struct {
	// FlagSet instance
	fs *flag.FlagSet
	// Source encoding report file to be parsed and used for sources for analysis
	flSrcReport string
	// Output directory for analysis results
	flOutDir string
}

// CreateAnalyseCommand will create Commander instace from AnalyseApp.
func CreateAnalyseCommand() Commander {
	longHelp := `Subcommand "analyse" will execute analysis stage on report generated from "encode"
stage. Report file is provided via -report flag and it is mandatory.

Examples:

  ease analyse -report encode_report.json -out-dir results`

	app := &AnalyseApp{
		fs: flag.NewFlagSet("analyse", flag.ContinueOnError),
	}
	app.fs.StringVar(&app.flSrcReport, "report", "", "Encoding report file as source for analysis (output from encoding stage)")
	app.fs.StringVar(&app.flOutDir, "out-dir", "", "Output directory to store results")
	app.fs.Usage = func() {
		printSubCommandUsage(longHelp, app.fs)
	}

	return app
}

func (a *AnalyseApp) Name() string {
	return a.fs.Name()
}

func (a *AnalyseApp) Help() {
	a.fs.Usage()
}

// init will do App state initialization.
func (a *AnalyseApp) init(args []string) error {
	if err := a.fs.Parse(args); err != nil {
		return &AppError{
			exitCode: 2,
			msg:      fmt.Sprintf("%s usage error", a.Name()),
		}
	}

	// If after flag parsing report file is not defined - error out.
	if a.flSrcReport == "" {
		a.Help()
		return &AppError{
			exitCode: 2,
			msg:      "mandatory option -report is missing",
		}
	}

	// If after flag parsing output directory is not defined - error out.
	if a.flOutDir == "" {
		a.Help()
		return &AppError{
			exitCode: 2,
			msg:      "mandatory option -out-dir is missing",
		}
	}

	// Report file should exist.
	if _, err := os.Stat(a.flSrcReport); err != nil {
		a.Help()
		return &AppError{
			exitCode: 2,
			msg:      fmt.Sprintf("report file does not exist? %s", err),
		}
	}

	return nil
}

func (a *AnalyseApp) Run(args []string) error {
	if err := a.init(args); err != nil {
		return err
	}

	// Check external tool dependencies - we require ffprobe to do bitrate calculations.
	if _, err := tools.FfprobePath(); err != nil {
		return &AppError{exitCode: 1, msg: fmt.Sprintf("dependency ffprobe: %s", err)}
	}

	// Read and parse report JSON file.
	logging.Debugf("Report JSON file %s", a.flSrcReport)
	r := parseReportFile(a.flSrcReport)

	// Extract data to work with.
	srcData := extractSourceData(r)
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

		if err := analysis.MultiPlotBitrate(compressedFile, bitratePlot); err != nil {
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

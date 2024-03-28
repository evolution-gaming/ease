// Copyright ©2022 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// ease tool's run subcommand implementation.

package main

import (
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/evolution-gaming/ease/internal/analysis"
	"github.com/evolution-gaming/ease/internal/encoding"
	"github.com/evolution-gaming/ease/internal/logging"
	"github.com/evolution-gaming/ease/internal/metric"
	"github.com/evolution-gaming/ease/internal/tools"
	"github.com/evolution-gaming/ease/internal/vqm"
	"github.com/jszwec/csvutil"
)

// CreateRunCommand will create instance of App.
func CreateRunCommand() *App {
	longHelp := `Subcommand "run" will execute encoding plan according to definition in file
provided as parameter to -plan flag and will calculate and report VQM
metrics. This flag is mandatory.

Examples:

  ease run -plan plan.json -out-dir path/to/output/dir`

	app := &App{
		fs:     flag.NewFlagSet("run", flag.ContinueOnError),
		gf:     globalFlags{},
		mStore: metric.NewStore(),
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
	// Encoding and VQ metric store
	mStore *metric.Store
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
func (a *App) encode(plan encoding.Plan) error {
	result, err := plan.Run()
	// Make sure to log any errors from RunResults.
	if ur := unrollResultErrors(result.RunResults); ur != "" {
		logging.Infof("Run had following ERRORS:\n%s", ur)
	}
	if err != nil {
		return fmt.Errorf("plan run: %w", err)
	}

	// Store encoding related metrics into mStore.
	for _, res := range result.RunResults {
		id := a.mStore.Insert(metric.Record{
			Name:             res.Name,
			SourceFile:       res.SourceFile,
			CompressedFile:   res.CompressedFile,
			Cmd:              res.Cmd,
			HStime:           res.Stats.HStime,
			HUtime:           res.Stats.HUtime,
			HElapsed:         res.Stats.HElapsed,
			Stime:            res.Stats.Stime,
			Utime:            res.Stats.Utime,
			Elapsed:          res.Stats.Elapsed,
			MaxRss:           res.Stats.MaxRss,
			VideoDuration:    res.VideoDuration,
			AvgEncodingSpeed: res.AvgEncodingSpeed,
		})
		logging.Debugf("Storing record (id=%v) with encoding metrics", id)
	}

	// Do VQM calculations for encoded videos.
	var vqmFailed bool = false
	for _, id := range a.mStore.GetIDs() {
		record, err := a.mStore.Get(id)
		if err != nil {
			vqmFailed = true
			logging.Infof("Error retrieving record from metric store: %s", err)
			continue
		}

		// Derive result file path.
		resFile := strings.TrimSuffix(record.CompressedFile, filepath.Ext(record.CompressedFile)) + "_vqm.json"
		// Create VMAF tool configuration.
		vmafCfg := vqm.FfmpegVMAFConfig{
			FfmpegPath:         a.cfg.FfmpegPath.Value(),
			LibvmafModelPath:   a.cfg.LibvmafModelPath.Value(),
			FfmpegVMAFTemplate: a.cfg.FfmpegVMAFTemplate.Value(),
			ResultFile:         resFile,
		}

		vqmTool, err2 := vqm.NewFfmpegVMAF(&vmafCfg, record.CompressedFile, record.SourceFile)
		if err2 != nil {
			vqmFailed = true
			logging.Infof("Error while initializing VQM tool: %s", err2)
			continue
		}

		logging.Infof("Start measuring VQMs for %s", record.CompressedFile)
		if err2 = vqmTool.Measure(); err2 != nil {
			vqmFailed = true
			logging.Infof("Failed calculate VQM for %s due to error: %s", record.CompressedFile, err2)
			continue
		}

		res, err2 := vqmTool.GetMetrics()
		if err2 != nil {
			vqmFailed = true
			logging.Infof("Error while getting metrics for %s: %s", record.CompressedFile, err2)
			continue
		}

		// Update record with VQ metrics.
		record.VQMResultFile = resFile
		record.PSNRMin = res.PSNR.Min
		record.PSNRMax = res.PSNR.Max
		record.PSNRMean = res.PSNR.Mean
		record.PSNRHarmonicMean = res.PSNR.HarmonicMean
		record.PSNRStDev = res.PSNR.StDev
		record.PSNRVariance = res.PSNR.Variance

		record.VMAFMin = res.VMAF.Min
		record.VMAFMax = res.VMAF.Max
		record.VMAFMean = res.VMAF.Mean
		record.VMAFHarmonicMean = res.VMAF.HarmonicMean
		record.VMAFStDev = res.VMAF.StDev
		record.VMAFVariance = res.VMAF.Variance

		record.MS_SSIMMin = res.MS_SSIM.Min
		record.MS_SSIMMax = res.MS_SSIM.Max
		record.MS_SSIMMean = res.MS_SSIM.Mean
		record.MS_SSIMHarmonicMean = res.MS_SSIM.HarmonicMean
		record.MS_SSIMStDev = res.MS_SSIM.StDev
		record.MS_SSIMVariance = res.MS_SSIM.Variance

		if err := a.mStore.Update(id, record); err != nil {
			vqmFailed = true
			logging.Infof("Error updating record (id=%v) for %s: %s", id, record.CompressedFile, err2)
			continue
		}
		logging.Debugf("Updating record (id=%v) with VQ metrics", id)
		logging.Infof("Done measuring VQMs for %s", record.CompressedFile)
	}

	if vqmFailed {
		return errors.New("VQM calculations had errors, see log for reasons")
	}

	return nil
}

// analyse will run analysis stage of plan execution.
func (a *App) analyse() error {
	for _, id := range a.mStore.GetIDs() {
		v, err := a.mStore.Get(id)
		if err != nil {
			return fmt.Errorf("fetching record by id (%v): %w", id, err)
		}
		// Create separate dir for results.
		base := path.Base(v.CompressedFile)
		base = strings.TrimSuffix(base, path.Ext(base))
		logging.Infof("Analysing %s", v.CompressedFile)
		resDir := path.Join(a.flOutDir, base)
		if err = os.MkdirAll(resDir, os.FileMode(0o755)); err != nil {
			return fmt.Errorf("creating directory: %w", err)
		}

		bitratePlot := path.Join(resDir, base+"_bitrate.png")
		vmafPlot := path.Join(resDir, base+"_vmaf.png")
		psnrPlot := path.Join(resDir, base+"_psnr.png")
		msssimPlot := path.Join(resDir, base+"_ms-ssim.png")

		// Need to get metadata of encoded video.
		meta, err := tools.FfprobeExtractMetadata(v.CompressedFile)
		if err != nil {
			return fmt.Errorf("extracting metadata: %w", err)
		}
		fps, err := parseFraction(meta.FrameRate)
		if err != nil {
			return fmt.Errorf("parsing frame rate: %w", err)
		}

		jsonFd, err := os.Open(v.VQMResultFile)
		if err != nil {
			return fmt.Errorf("opening VQM file: %w", err)
		}

		var frameMetrics vqm.FrameMetrics
		err = frameMetrics.FromFfmpegVMAF(jsonFd)
		// Close jsonFd file descriptor at earliest convenience. Should avoid use of defer
		// in loop in this case.
		jsonFd.Close()
		if err != nil {
			return fmt.Errorf("failed converting to FrameMetrics: %w", err)
		}

		size := len(frameMetrics)
		vmafs := make(metricXYs, 0, size)
		psnrs := make(metricXYs, 0, size)
		msssims := make(metricXYs, 0, size)
		for _, v := range frameMetrics {
			// Calculate timestamp for given frame.
			ts := float64(v.FrameNum) / fps
			vmafs = append(vmafs, metricXY{X: ts, Y: v.VMAF})
			psnrs = append(psnrs, metricXY{X: ts, Y: v.PSNR})
			msssims = append(msssims, metricXY{X: ts, Y: v.MS_SSIM})
		}

		// Since frameMetrics coming from JSON can be absent, we check for this case, e.g.
		// if all metric values are 0 then most probable case is that metric was missing
		// from source JSON. This is due to how unmarshaling works in Go.
		yIsZero := func(x metricXY) bool { return x.Y == 0 }
		skipVMAF := all(vmafs, yIsZero)
		skipPSNR := all(psnrs, yIsZero)
		skipMSSSIM := all(msssims, yIsZero)

		if err := analysis.MultiPlotBitrate(v.CompressedFile, bitratePlot, a.cfg.FfprobePath.Value()); err != nil {
			return fmt.Errorf("creating bitrate plot: %w", err)
		}
		logging.Infof("Bitrate plot done: %s", bitratePlot)

		if skipVMAF {
			logging.Info("Skip VMAF multi-plot, metric missing")
		} else {
			if err := analysis.MultiPlotVqm(vmafs, "VMAF", base, vmafPlot); err != nil {
				return fmt.Errorf("creating VMAF multiplot: %w", err)
			}
			logging.Infof("VMAF multi-plot done: %s", vmafPlot)
		}

		if skipPSNR {
			logging.Info("Skip PSNR multi-plot, metric missing")
		} else {
			if err := analysis.MultiPlotVqm(psnrs, "PSNR", base, psnrPlot); err != nil {
				return fmt.Errorf("creating PSNR multiplot: %w", err)
			}
			logging.Infof("PSNR multi-plot done: %s", psnrPlot)
		}

		if skipMSSSIM {
			logging.Info("Skip MS-SSIM multi-plot, metric missing")
		} else {
			if err := analysis.MultiPlotVqm(msssims, "MS-SSIM", base, msssimPlot); err != nil {
				return fmt.Errorf("creating MS-SSIM multiplot: %w", err)
			}
			logging.Infof("MS-SSIM multi-plot done: %s", msssimPlot)
		}
	}

	return nil
}

// saveReport writes recorded metrics to report file.
func (a *App) saveReport() error {
	ids := a.mStore.GetIDs()
	report := make([]metric.Record, 0, len(ids))
	for _, id := range ids {
		r, err := a.mStore.Get(id)
		if err != nil {
			return fmt.Errorf("getting record (id=%v) from metric store: %w", id, err)
		}
		report = append(report, r)
	}

	reportPath := path.Join(a.flOutDir, a.cfg.ReportFileName.Value())
	reportOut, err := os.Create(reportPath)
	if err != nil {
		return fmt.Errorf("creating CSV report file: %w", err)
	}
	defer reportOut.Close()

	w := csv.NewWriter(reportOut)
	if err := csvutil.NewEncoder(w).Encode(report); err != nil {
		return fmt.Errorf("writing CSV report: %w", err)
	}
	w.Flush()

	return nil
}

// Run is main entry point into App execution.
func (a *App) Run(args []string) error {
	logging.Infof("ease version: %s", vInfo)
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

	// To avoid ambiguity, resolve output path to absolute representation.
	outDirPath, err := filepath.Abs(a.flOutDir)
	if err != nil {
		return &AppError{exitCode: 1, msg: err.Error()}
	}
	plan := encoding.NewPlan(pc, outDirPath)

	// Early return in "dry run" mode.
	if a.flDryRun {
		logging.Info("Dry run mode finished!")
		return nil
	}

	// Run encode stage.
	if err = a.encode(plan); err != nil {
		return &AppError{exitCode: 1, msg: err.Error()}
	}

	// Save report.
	if err = a.saveReport(); err != nil {
		return &AppError{exitCode: 1, msg: err.Error()}
	}

	// Run analysis stage.
	if err = a.analyse(); err != nil {
		return &AppError{exitCode: 1, msg: err.Error()}
	}

	logging.Info("Done")
	return nil
}

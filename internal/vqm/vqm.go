// Copyright ©2022 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Contains implementation of VQM tool that uses ffmpeg and libvmaf along with
// related data structures and interfaces.

package vqm

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"text/template"

	"github.com/evolution-gaming/ease/internal/logging"
	"github.com/evolution-gaming/ease/internal/tools"
	"github.com/google/shlex"
	"gonum.org/v1/gonum/floats"
	"gonum.org/v1/gonum/stat"
)

var DefaultFfmpegVMAFTemplate = "-hide_banner -i {{.CompressedFile}} -i {{.SourceFile}} " +
	"-lavfi libvmaf=n_subsample=1:log_path={{.ResultFile}}:feature=name=psnr:" +
	"log_fmt=json:model=path={{.ModelPath}}:n_threads={{.NThreads}} -f null -"

// FfmpegVMAFConfig exposes parameters for ffmpegVMAF creation.
type FfmpegVMAFConfig struct {
	FfmpegPath         string
	LibvmafModelPath   string
	FfmpegVMAFTemplate string
	ResultFile         string
}

// NewFfmpegVMAF will initialize VQM Measurer based on ffmpeg and libvmaf.
func NewFfmpegVMAF(cfg *FfmpegVMAFConfig, compressedFile, sourceFile string) (*FfmpegVMAF, error) {
	var vqt *FfmpegVMAF

	// Too much CPU threads are also bad. This was an issue on 128 threaded AMD
	// EPYC, ffmpeg was deadlocking at some point during VMAF calculations.
	nThreads := 32

	if runtime.NumCPU() < nThreads {
		nThreads = runtime.NumCPU()
	}

	// Template requires a struct with exported fields.
	tplContext := struct {
		SourceFile     string
		CompressedFile string
		ResultFile     string
		ModelPath      string
		NThreads       int
	}{
		SourceFile:     sourceFile,
		CompressedFile: compressedFile,
		ResultFile:     cfg.ResultFile,
		ModelPath:      cfg.LibvmafModelPath,
		NThreads:       nThreads,
	}

	var cmd strings.Builder
	tpl := template.Must(template.New("ffmpeg").Parse(cfg.FfmpegVMAFTemplate))
	err := tpl.Execute(&cmd, tplContext)
	if err != nil {
		return vqt, fmt.Errorf("NewFfmpegVMAF() execute template: %w", err)
	}
	ffmpegArgs, err := shlex.Split(cmd.String())
	if err != nil {
		return vqt, fmt.Errorf("NewFfmpegVMAF() prepare command: %w", err)
	}

	vqt = &FfmpegVMAF{
		exePath:        cfg.FfmpegPath,
		ffmpegArgs:     ffmpegArgs,
		sourceFile:     sourceFile,
		compressedFile: compressedFile,
		resultFile:     cfg.ResultFile,
		output:         []byte{},
		measured:       false,
	}

	return vqt, nil
}

// FfmpegVMAF defines VQM tool and implements Measurer interface.
type FfmpegVMAF struct {
	// Path to ffmpeg executable
	exePath string
	// ffmpeg command arguments
	ffmpegArgs []string
	// Uncompressed source file
	sourceFile string
	// Compressed file that will be compared to sourceFile
	compressedFile string
	// ffmpeg generated results wil be stored in this file
	resultFile string
	output     []byte
	measured   bool
}

func (f *FfmpegVMAF) Measure() error {
	var err error

	if f.measured {
		return errors.New("Measure() already executed")
	}

	// First we should check if source and compressed files have equal number of
	// frames, if it is not the case - then VQM will be off.
	srcMeta, err := tools.FfprobeExtractMetadata(f.sourceFile)
	if err != nil {
		return fmt.Errorf("source file metadata: %w", err)
	}
	compressedMeta, err := tools.FfprobeExtractMetadata(f.compressedFile)
	if err != nil {
		return fmt.Errorf("compressed file metadata: %w", err)
	}
	if srcMeta.FrameCount != compressedMeta.FrameCount {
		return fmt.Errorf("frame count mismatch: source %v != compressed %v", srcMeta.FrameCount, compressedMeta.FrameCount)
	}

	cmd := exec.Command(f.exePath, f.ffmpegArgs...) //#nosec G204
	logging.Debugf("VQM tool command: %v", cmd.Args)
	f.output, err = cmd.CombinedOutput()
	if err != nil {
		logging.Infof("VQM tool execution failure:\n%s", cmd.String())
		logging.Infof("VQM tool output:\n%s", f.output)
		return fmt.Errorf("VQM calculation error: %w", err)
	}

	f.measured = true
	return nil
}

type AggregateMetric struct {
	VMAF    Metric
	PSNR    Metric
	MS_SSIM Metric
}

type Metric struct {
	Mean         float64
	HarmonicMean float64
	Min          float64
	Max          float64
	StDev        float64
	Variance     float64
}

func (f *FfmpegVMAF) GetMetrics() (*AggregateMetric, error) {
	if !f.measured {
		return nil, errors.New("GetMetrics() depends on Measure() called first")
	}

	am := &AggregateMetric{}
	// Unmarshal metrics from result file.
	j, err := os.Open(f.resultFile)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}

	var metrics FrameMetrics
	err2 := metrics.FromFfmpegVMAF(j)
	if err2 != nil {
		return nil, fmt.Errorf("parsing JSON: %w", err2)
	}

	// Convert to vectors to apply aggregations.
	m := struct {
		VMAF    []float64
		PSNR    []float64
		MS_SSIM []float64
	}{}
	for _, v := range metrics {
		m.VMAF = append(m.VMAF, v.VMAF)
		m.PSNR = append(m.PSNR, v.PSNR)
		m.MS_SSIM = append(m.MS_SSIM, v.MS_SSIM)
	}

	am.VMAF.Min = floats.Min(m.VMAF)
	am.VMAF.Max = floats.Max(m.VMAF)
	am.VMAF.HarmonicMean = stat.HarmonicMean(m.VMAF, nil)
	am.VMAF.Variance = stat.Variance(m.VMAF, nil)
	am.VMAF.Mean, am.VMAF.StDev = stat.MeanStdDev(m.VMAF, nil)

	am.PSNR.Min = floats.Min(m.PSNR)
	am.PSNR.Max = floats.Max(m.PSNR)
	am.PSNR.HarmonicMean = stat.HarmonicMean(m.PSNR, nil)
	am.PSNR.Variance = stat.Variance(m.PSNR, nil)
	am.PSNR.Mean, am.PSNR.StDev = stat.MeanStdDev(m.PSNR, nil)

	am.MS_SSIM.Min = floats.Min(m.MS_SSIM)
	am.MS_SSIM.Max = floats.Max(m.MS_SSIM)
	am.MS_SSIM.HarmonicMean = stat.HarmonicMean(m.MS_SSIM, nil)
	am.MS_SSIM.Variance = stat.Variance(m.MS_SSIM, nil)
	am.MS_SSIM.Mean, am.MS_SSIM.StDev = stat.MeanStdDev(m.MS_SSIM, nil)

	return am, nil
}

// This and following are helper structs for libvmaf JSON result.
type ffmpegVMAFResult struct {
	Version       string        `json:"version"`
	Frames        []frame       `json:"frames"`
	PooledMetrics pooledMetrics `json:"pooled_metrics"`
}

type frame struct {
	FrameNum uint   `json:"frameNum"`
	Metrics  metric `json:"metrics"`
}

type metric struct {
	VMAF    float64
	PSNR    float64
	MS_SSIM float64
}

// UnmarshalJSON implements json.Unmarshaler interface for metric.
//
// A custom unmarshaler is needed to work around lack of stability around libvmaf measured
// VQ metric field names in output.
func (m *metric) UnmarshalJSON(b []byte) error {
	// Ignore "null" as per convention.
	if string(b) == "null" {
		return nil
	}

	// Unmarshal JSON blob into map so that it includes all fields.
	raw := make(map[string]float64)
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}

	// Pull out required fields and in some cases their aliases (due to libvmaf changing
	// field names at will)
	for k, v := range raw {
		switch k {
		case "vmaf":
			m.VMAF = v
		case "psnr", "psnr_y":
			m.PSNR = v
		case "ms_ssim", "float_ms_ssim":
			m.MS_SSIM = v
		}
	}

	return nil
}

type pooledMetrics struct {
	VMAF    pMetric
	PSNR    pMetric
	MS_SSIM pMetric
}

// UnmarshalJSON implements json.Unmarshaler interface for pooledMetrics.
//
// A custom unmarshaler is needed to work around lack of stability around libvmaf measured
// VQ metric field names in output.
func (p *pooledMetrics) UnmarshalJSON(b []byte) error {
	// Ignore "null" as per convention.
	if string(b) == "null" {
		return nil
	}

	raw := make(map[string]pMetric)
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}

	for k, v := range raw {
		switch k {
		case "vmaf":
			p.VMAF = v
		case "psnr", "psnr_y":
			p.PSNR = v
		case "ms_ssim", "float_ms_ssim":
			p.MS_SSIM = v
		}
	}
	return nil
}

type pMetric struct {
	Min          float64 `json:"min"`
	Max          float64 `json:"max"`
	Mean         float64 `json:"mean"`
	HarmonicMean float64 `json:"harmonic_mean"`
}

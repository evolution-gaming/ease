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
	"github.com/google/shlex"
)

var DefaultFfmpegVMAFTemplate = "-hide_banner -i {{.CompressedFile}} -i {{.SourceFile}} " +
	"-lavfi libvmaf=n_subsample=1:log_path={{.ResultFile}}:psnr=1:log_fmt=json:model_path={{.ModelPath}}:n_threads={{.NThreads}} -f null -"

// Measurer is an interface that must be implemented by VQM tool which is capable of
// calculating Vide Quality Metrics.
type Measurer interface {
	// Measure should run actual VQM measuring process
	Measure() error
	// GetResult will retrieve VQM measurement Result
	GetResult() (Result, error)
}

// Result represents Measurer tool execution result.
type Result struct {
	SourceFile     string
	CompressedFile string
	ResultFile     string
	Metrics        VideoQualityMetrics
}

// VideoQualityMetrics is a struct of meaningful Video Quality Metrics.
type VideoQualityMetrics struct {
	PSNR    float64
	MS_SSIM float64
	VMAF    float64
}

// FfmpegVMAFConfig exposes parameters for ffmpegVMAF creation.
type FfmpegVMAFConfig struct {
	FfmpegPath         string
	LibvmafModelPath   string
	FfmpegVMAFTemplate string
	ResultFile         string
}

// NewFfmpegVMAF will initialize VQM Measurer based on ffmpeg and libvmaf.
func NewFfmpegVMAF(cfg *FfmpegVMAFConfig, compressedFile, sourceFile string) (Measurer, error) {
	var vqt *ffmpegVMAF

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

	vqt = &ffmpegVMAF{
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

// ffmpegVMAF defines VQM tool and implements Measurer interface.
type ffmpegVMAF struct {
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

func (f *ffmpegVMAF) Measure() error {
	if f.measured {
		return errors.New("Measure() already executed")
	}
	cmd := exec.Command(f.exePath, f.ffmpegArgs...) //#nosec G204
	logging.Debugf("VQM tool command: %v", cmd.Args)
	var err error
	f.output, err = cmd.CombinedOutput()
	if err != nil {
		logging.Infof("VQM tool execution failure:\n%s", cmd.String())
		logging.Infof("VQM tool output:\n%s", f.output)
		return fmt.Errorf("VQM calculation error: %w", err)
	}
	f.measured = true
	return nil
}

func (f *ffmpegVMAF) GetResult() (Result, error) {
	var vqr Result

	// Depend on Measure() being executed.
	if !f.measured {
		return vqr, errors.New("GetResult() depends on Measure() called first")
	}

	resData, err := os.ReadFile(f.resultFile)
	if err != nil {
		return vqr, fmt.Errorf("VideoQualityTool.GetResult() reading %s: %w", f.resultFile, err)
	}
	vqm, err := f.unmarshalResultJSON(resData)
	if err != nil {
		return vqr, fmt.Errorf("VideoQualityTool.GetResult() in resultParser(): %w", err)
	}
	vqr = Result{
		Metrics:        vqm,
		SourceFile:     f.sourceFile,
		CompressedFile: f.compressedFile,
		ResultFile:     f.resultFile,
	}
	return vqr, nil
}

// unmarshalResultJSON will unmarshal libvmaf JSON result to VideoQualityMetrics.
func (f *ffmpegVMAF) unmarshalResultJSON(data []byte) (VideoQualityMetrics, error) {
	var vqm VideoQualityMetrics
	res := &ffmpegVMAFResult{}

	if err := json.Unmarshal(data, res); err != nil {
		return vqm, fmt.Errorf("parseResult() unmarshal JSON: %w", err)
	}
	vqm = VideoQualityMetrics{
		VMAF:    res.PooledMetrics.VMAF.Mean,
		PSNR:    res.PooledMetrics.PSNR.Mean,
		MS_SSIM: res.PooledMetrics.MS_SSIM.Mean,
	}
	return vqm, nil
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

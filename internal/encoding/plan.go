// Copyright ©2022 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package encoding

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/evolution-gaming/ease/internal/logging"
	"github.com/evolution-gaming/ease/internal/lw"
	"github.com/evolution-gaming/ease/internal/tools"
)

const (
	inputPlaceholder   = "%INPUT%"
	outputPlaceholder  = "%OUTPUT%"
	logFilePlaceholder = "%LOGFILE%"
	outputBufferSize   = 5 * 1024 * 1024 // 5 MiB for output buffer
)

// EncoderCmd defines an encoder command struct.
type EncoderCmd struct {
	// Name of encoding
	Name string
	// SourceFile is uncompressed video source aka mezzanine file
	SourceFile string
	// CompressedFile is a result of compressing InputFile
	CompressedFile string
	// OutputFile contains text output generated from encoder (log output)
	OutputFile string
	// LogFile is additional log file that encoder may generate
	LogFile string
	// WorkDir contains current working directory (a.k.a CWD, PWD)
	WorkDir string
	// Cmd is a actual "executable" encoder commandline with parameters
	Cmd string
}

// Run will run all encoding commands defined for this Plan.
//
// Error is nil if all encoding commands succeed without errors.
//
// TODO: After refactoring lost the ability to control output io.Writer. Maybe
// need to add option to pass in own io.Writer (for testing purposes?)
func (s *EncoderCmd) Run() RunResult {
	// Initialize RunResult from "this" EncoderCmd.
	r := RunResult{EncoderCmd: *s}

	// Backing buffer for stderr.
	var buf bytes.Buffer
	var outWriter, memWriter io.Writer
	// Explicitly limit stderr buffer to certain size to protect ourselves
	// from some runaway process flooding output.
	memWriter = lw.LimitWriter(&buf, outputBufferSize)

	f, err := os.Create(s.OutputFile)
	if err != nil {
		logging.Infof("Unable to redirect output to file: %s", err)
		r.AddError(err)
		return r
	} else {
		logging.Infof("Output redirected to file: %s", f.Name())
		outWriter = io.MultiWriter(memWriter, f)
		defer f.Close()
	}

	// Encoder commands come in different flavours and in some cases commands
	// can make use various commands connected via pipes, this can be supported
	// by employing shell to execute commands.We trust user to provide safe
	// encoder command, otherwise this can be a security issue.
	r.cmd = exec.Command("sh", "-c", s.Cmd) //#nosec G204
	// Explicitly limit stderr buffer to certain size to protect ourselves
	// from some runaway process flooding output.
	r.cmd.Stderr = outWriter
	// Time executions to calculate a wall time.
	start := time.Now()
	if err = r.cmd.Run(); err != nil {
		logging.Infof("Run error for %s: %s", r.Name, err)
		logging.Debugf("Command: %s", r.cmd)
		logging.Debugf("Stderr: %s", buf.Bytes())
		r.AddError(err)
	}
	r.Stats = NewUsageStat(time.Since(start), r.Rusage())
	// Add VideoDuration and also calculate approximation to average encoding speed.
	vmeta, err := tools.FfprobeExtractMetadata(r.CompressedFile)
	if err != nil {
		logging.Infof("Unable to query compressed video metadata: %v", err)
		r.AddError(err)
	} else {
		r.VideoDuration = vmeta.Duration
		r.AvgEncodingSpeed = vmeta.Duration / r.Stats.Elapsed.Seconds()
	}
	r.stderr = buf.Bytes()

	return r
}

// Scheme is an encoder string with input and output placeholders.
//
// For now it is just an encoding command line string with placeholders for input
// and output file. Like ffmpeg command with flags.
//
// A Name field will be used when generating output file, so use it sensibly -
// think of it as as part of some nomenclature scheme.
type Scheme struct {
	Name       string
	CommandTpl string
}

// UnmarshalJSON implement Unmarshaler interface for Scheme type.
func (s *Scheme) UnmarshalJSON(data []byte) error {
	// Since JSON Scheme.CommandTpl is a string array we create a "temporary"
	// struct that will be used to decode JSON, we will use this struct to
	// construct Scheme fields.
	scheme := struct {
		Name       string
		CommandTpl []string
	}{}
	if err := json.Unmarshal(data, &scheme); err != nil {
		return err
	}
	s.Name = scheme.Name
	// This is the part that needed the whole custom Unmarshaler for Scheme struct.
	s.CommandTpl = strings.Join(scheme.CommandTpl, "")

	return nil
}

// Expand will generate complete encoding commands based on provided "context".
//
// "Context" being input/source files and output directory.
//
// TODO: Not sure about the name Expand(). Also, function body looks busy.
func (s *Scheme) Expand(sourceFiles []string, outDir string) (cmds []EncoderCmd) {
	for _, sFile := range sourceFiles {
		oFileBase := generateOutputFileNameBase(sFile, outDir, s.Name)

		// Determine compressed file extension (including the dot).
		var compressedFileExt string
		extMatcher := regexp.MustCompile(fmt.Sprintf(`%s(\.\w+)*`, outputPlaceholder))
		m := extMatcher.FindStringSubmatch(string(s.CommandTpl))
		if m != nil {
			compressedFileExt = m[1]
		}

		// Generate various filenames for later use.
		compressedFile := fmt.Sprintf("%s%s", oFileBase, compressedFileExt)
		outputFile := fmt.Sprintf("%s.out", oFileBase)
		logFile := fmt.Sprintf("%s.log", oFileBase)

		// Replace placeholders in command template.
		cmdStr := strings.ReplaceAll(string(s.CommandTpl), inputPlaceholder, sFile)
		cmdStr = strings.ReplaceAll(cmdStr, outputPlaceholder, oFileBase)
		cmdStr = strings.ReplaceAll(cmdStr, logFilePlaceholder, logFile)

		cwd, err := os.Getwd()
		if err != nil {
			logging.Infof("Expand() unable to get working directory: %s", err)
		}

		if err != nil {
			logging.Infof("Expand() error on commandline %s: %s", cmdStr, err)
			continue
		}

		ec := EncoderCmd{
			Name:           s.Name,
			SourceFile:     sFile,
			CompressedFile: compressedFile,
			OutputFile:     outputFile,
			LogFile:        logFile,
			WorkDir:        cwd,
			Cmd:            cmdStr,
		}
		cmds = append(cmds, ec)
	}

	return cmds
}

type Plan struct {
	// Embed PlanConfig struct
	PlanConfig
	// Executable encoder commands
	Commands []EncoderCmd
	// Output directory
	OutDir string
	// Flag to signal if output dir has been created
	outDirCreated bool
}

// NewPlan will create Plan instance from given PlanConfig.
func NewPlan(pc PlanConfig, outDir string) Plan {
	p := Plan{
		PlanConfig:    pc,
		OutDir:        outDir,
		outDirCreated: false,
	}
	for _, scheme := range p.Schemes {
		cmds := scheme.Expand(p.Inputs, p.OutDir)
		p.Commands = append(p.Commands, cmds...)
	}
	return p
}

// Run executes encoding commands part of this Plan.
func (s *Plan) Run() (PlanResult, error) {
	var runError error
	result := PlanResult{
		StartTime:  time.Now(),
		RunResults: make([]RunResult, len(s.Commands)),
	}

	// Start by creating output dir s.OutDir.
	if err := s.ensureOutDir(); err != nil {
		return result, err
	}

	for i := range s.Commands {
		logging.Infof("Start encoding %s -> %s", s.Commands[i].SourceFile, s.Commands[i].CompressedFile)
		result.RunResults[i] = s.Commands[i].Run()
		logging.Infof("Done encoding %s -> %s", s.Commands[i].SourceFile, s.Commands[i].CompressedFile)
	}
	result.EndTime = time.Now()

	for i := range result.RunResults {
		if len(result.RunResults[i].Errors) != 0 {
			runError = errors.New("Plan run executed with errors")
		}
	}
	return result, runError
}

// ensureOutDir will create output directory if it does not exist.
func (p *Plan) ensureOutDir() error {
	if p.outDirCreated {
		return nil
	}
	logging.Debugf("Creating output directory: %s", p.OutDir)
	err := os.MkdirAll(p.OutDir, os.FileMode(0o775))
	if err != nil {
		return fmt.Errorf("ensureOutDir(): %w", err)
	}
	p.outDirCreated = true
	return nil
}

// PlanResult holds Plan execution result state.
type PlanResult struct {
	StartTime  time.Time
	EndTime    time.Time
	RunResults []RunResult
}

// RunResult contains a status of a single encoding run.
type RunResult struct {
	EncoderCmd
	Errors           []error
	cmd              *exec.Cmd
	stderr           []byte
	Stats            UsageStat
	VideoDuration    float64
	AvgEncodingSpeed float64
}

// ExitCode returns exit code of executed encoding run.
func (s *RunResult) ExitCode() int {
	return s.cmd.ProcessState.ExitCode()
}

// Output returns output from encoding run.
//
// For encoders Stdout is usually reserved for piping encoded stream, so
// diagnostics output usually goes to Stderr.
func (s *RunResult) Output() string {
	return string(s.stderr)
}

func (s *RunResult) Rusage() *syscall.Rusage {
	usage, _ := s.cmd.ProcessState.SysUsage().(*syscall.Rusage)
	return usage
}

func (s *RunResult) AddError(e error) {
	s.Errors = append(s.Errors, e)
}

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

// generateOutputFileNameBase will generate a sensible output filename without extension.
func generateOutputFileNameBase(inputFile, outDir, postfix string) string {
	// Normalize filename strings to sane format (no spaces).
	normalize := func(input string) string {
		return strings.ReplaceAll(input, " ", "_")
	}
	baseName := path.Base(inputFile)
	baseName = strings.TrimSuffix(baseName, filepath.Ext(baseName))
	return path.Join(outDir, fmt.Sprintf("%s_%s", normalize(baseName), normalize(postfix)))
}

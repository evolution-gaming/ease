// Copyright ©2022 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Application configuration structures.

package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/evolution-gaming/ease/internal/logging"
	"github.com/evolution-gaming/ease/internal/tools"
	"github.com/evolution-gaming/ease/internal/vqm"
)

var (
	ErrInvalidConfig  = errors.New("invalid configuration")
	defaultReportFile = "report.json"
)

// Config represent application configuration.
type Config struct {
	FfmpegPath         ConfigVal[string] `json:"ffmpeg_path,omitempty"`
	FfprobePath        ConfigVal[string] `json:"ffprobe_path,omitempty"`
	LibvmafModelPath   ConfigVal[string] `json:"libvmaf_model_path,omitempty"`
	FfmpegVMAFTemplate ConfigVal[string] `json:"ffmpeg_vmaf_template,omitempty"`
	ReportFileName     ConfigVal[string] `json:"report_file_name,omitempty"`
}

// Verify will check that configuration is valid.
//
// Will check that configuration option values are sensible.
func (c *Config) Verify() error {
	msgs := []string{}
	// Check that ffmpeg exists.
	if !fileExists(c.FfmpegPath.Value()) {
		msgs = append(msgs, "invalid ffmpeg path")
	}
	// Check that ffprobe exists.
	if !fileExists(c.FfprobePath.Value()) {
		msgs = append(msgs, "invalid ffprobe path")
	}
	// Check that libvmaf model file exists.
	if !fileExists(c.LibvmafModelPath.Value()) {
		msgs = append(msgs, "invalid libvmaf model file path")
	}
	// Template should not be nil.
	if c.FfmpegVMAFTemplate.IsNil() {
		msgs = append(msgs, "empty ffmpeg VMAF template")
	}
	// Report file should not be nil.
	if c.ReportFileName.IsNil() {
		msgs = append(msgs, "empty report file name")
	}

	if len(msgs) != 0 {
		return fmt.Errorf("%s: %w", strings.Join(msgs, ", "), ErrInvalidConfig)
	}
	return nil
}

// OverrideFrom will overwrite fields from given Config object.
//
// Only fields that are "not-nil" (as per IsNil() method) in src Config object will be
// overwritten.
func (c *Config) OverrideFrom(src Config) {
	// TODO: some way to iterate over fields and set them (reflection?) otherwise need to
	// remember to update this method when new  fields are added.
	if !src.FfmpegPath.IsNil() {
		c.FfmpegPath = src.FfmpegPath
	}
	if !src.FfprobePath.IsNil() {
		c.FfprobePath = src.FfprobePath
	}
	if !src.LibvmafModelPath.IsNil() {
		c.LibvmafModelPath = src.LibvmafModelPath
	}
	if !src.FfmpegVMAFTemplate.IsNil() {
		c.FfmpegVMAFTemplate = src.FfmpegVMAFTemplate
	}
	if !src.ReportFileName.IsNil() {
		c.ReportFileName = src.ReportFileName
	}
}

// loadDefaultConfig will create a default configuration.
//
// For some configuration options a default value will be specified, for others an
// auto-detection mechanism will populate option values.
func loadDefaultConfig() (Config, error) {
	var cfg Config

	// For default configuration attempt to locate ffmpeg binary.
	ffmpeg, err := tools.FfmpegPath()
	if err != nil {
		return cfg, fmt.Errorf("DefaultConfig: %w", err)
	}

	// For default configuration attempt to locate ffprobe binary.
	ffprobe, err := tools.FfprobePath()
	if err != nil {
		return cfg, fmt.Errorf("DefaultConfig: %w", err)
	}

	// For default configuration attempt to locate VMAF model file.
	libvmafModel, err := tools.FindLibvmafModel()
	if err != nil {
		return cfg, fmt.Errorf("DefaultConfig: %w", err)
	}

	cfg = Config{
		FfmpegPath:         NewConfigVal(ffmpeg),
		FfprobePath:        NewConfigVal(ffprobe),
		LibvmafModelPath:   NewConfigVal(libvmafModel),
		FfmpegVMAFTemplate: NewConfigVal(vqm.DefaultFfmpegVMAFTemplate),
		ReportFileName:     NewConfigVal(defaultReportFile),
	}

	return cfg, nil
}

// loadConfigFromFile will load configuration from file.
//
// Only JSON is supported at this point.
func loadConfigFromFile(f string) (cfg Config, err error) {
	fileExt := strings.ToLower(filepath.Ext(f))
	switch fileExt {
	case ".json":
		return loadJSON(f)
	default:
		return cfg, fmt.Errorf("unknown config format: %s", fileExt)
	}
}

// LoadConfig will return merged default config and config from file. This is main
// function to use for config loading. Configuration file is optional e.g. can be "".
func LoadConfig(configFile string) (cfg Config, err error) {
	// Initialize default configuration.
	cfg, err = loadDefaultConfig()
	if err != nil {
		return cfg, err
	}

	// Load configuration from file and override default configuration options.
	if configFile != "" {
		c, err := loadConfigFromFile(configFile)
		if err != nil {
			return cfg, err
		}
		// Configuration file can specify full set or partial set of configuration
		// options. So we only want to override those options that have been specified in
		// config file, re st will remain as per default config.
		cfg.OverrideFrom(c)
	}

	return cfg, nil
}

func loadJSON(f string) (cfg Config, err error) {
	b, err := os.ReadFile(f)
	if err != nil {
		return cfg, fmt.Errorf("config from JSON file: %w", err)
	}

	if len(b) == 0 {
		return cfg, fmt.Errorf("JSON file is empty: %w", ErrInvalidConfig)
	}

	if err = json.Unmarshal(b, &cfg); err != nil {
		return cfg, fmt.Errorf("config from JSON document: %w", err)
	}

	return cfg, nil
}

// In order to support Config overriding we have to implement wrapper type for Config
// fields. Otherwise it is hard to distinguish skipped fields, for instance when loading
// partial configuration from file: in that case it would be impossible to  distinguish
// between say string fields zero value and empty string values as explicitly specified in
// configuration file.

// NewConfigVal is constructor for ConfigVal. It will wrap its argument into ConfigVal.
func NewConfigVal[T any](v T) ConfigVal[T] {
	return ConfigVal[T]{v: &v}
}

// ConfigVal is a wrapper for Config field value.
type ConfigVal[T any] struct {
	// Store wrapped value as pointer in order to have ability to distinguish between
	// unspecified ConfigVal and a value that is the same as zero value for wrapped type.
	// In this case a zero value for pointer is nil.
	//
	// For example a zero value for string is "" which is impossible to distinguish from
	// explicit empty string "".
	v *T
}

// Value will return wrapped value.
//
// In case field has not been defined e.g. is zero value, then appropriate zero value of
// wrapped typw will be returned.
func (o *ConfigVal[T]) Value() T {
	if o.IsNil() {
		var v T
		return v
	}
	return *o.v
}

// IsNil check if wrapped value is nil.
func (o *ConfigVal[T]) IsNil() bool {
	// Zero value for pointer type is nil.
	return o.v == nil
}

// UnmarshalJSON implements json.Unmarshaler interface for ConfigVal.
func (o *ConfigVal[T]) UnmarshalJSON(b []byte) error {
	var val T
	err := json.Unmarshal(b, &val)
	if err != nil {
		return err
	}
	o.v = &val
	return nil
}

// MarshalJSON implements json.Marshaler interface for ConfigVal.
func (o ConfigVal[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(o.Value())
}

func CreateDumpConfCommand() Commander {
	longHelp := `Command "dump-conf" will print actual application configuration taking into account
configuration file provided and default configuration values.

Examples:

	ease dump-conf
	ease dump-conf -conf path/to/config.json`

	app := &DumpConfApp{
		fs:  flag.NewFlagSet("dump-conf", flag.ContinueOnError),
		gf:  globalFlags{},
		out: os.Stdout,
	}
	app.gf.Register(app.fs)
	app.fs.Usage = func() {
		printSubCommandUsage(longHelp, app.fs)
	}

	return app
}

// Also define command "dump-conf" here.

// Make sure App implements Commander interface.
var _ Commander = (*DumpConfApp)(nil)

// DumpConfApp is subcommand application context that implements Commander interface.
// Although this is very simple application, but for consistency sake is is implemented in
// similar style as other subcommands.
type DumpConfApp struct {
	out io.Writer
	fs  *flag.FlagSet
	gf  globalFlags
}

// Run is main entry point into BitrateApp execution.
func (d *DumpConfApp) Run(args []string) error {
	if err := d.fs.Parse(args); err != nil {
		return &AppError{
			exitCode: 2,
			msg:      "usage error",
		}
	}

	if d.gf.Debug {
		logging.EnableDebugLogger()
	}

	// Load application configuration.
	cfg, err := LoadConfig(d.gf.ConfFile)
	if err != nil {
		return &AppError{exitCode: 1, msg: err.Error()}
	}

	enc := json.NewEncoder(d.out)
	enc.SetIndent("", "  ")
	if err := enc.Encode(cfg); err != nil {
		return &AppError{exitCode: 1, msg: err.Error()}
	}

	// Also, report if configuration is valid.
	if err := cfg.Verify(); err != nil {
		return &AppError{exitCode: 1, msg: fmt.Sprintf("configuration validation: %s", err)}
	}

	return nil
}

// Copyright ©2022 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Application Config related tests.
package main

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"reflect"
	"strings"
	"testing"

	"github.com/evolution-gaming/ease/internal/it"
)

func Test_loadDefaultConfig(t *testing.T) {
	c := loadDefaultConfig()
	err := c.Verify()
	it.Should(t, err == nil, "DefaultConfig should be valid: got = %v, want = nil", err)
}

func Test_loadDefaultConfig_Negative(t *testing.T) {
	// Messing up PATH should result in failure detecting ffmpeg and ffprobe which
	// should result in error from calling DefaultConfig().
	t.Setenv("PATH", "")
	c := loadDefaultConfig()
	it.Should(t, c.Verify() != nil, "Expected error, got nil")
}

func Test_loadConfigFile(t *testing.T) {
	// For this case we do not strictly need config that is valid as per Config.Verify(),
	// just verify that loading configuration from file works.
	tests := map[string]struct {
		want  Config
		given []byte
	}{
		"Full": {
			given: []byte(`{
				"ffmpeg_path": "test_ffmpeg",
				"ffprobe_path": "test_ffprobe",
				"ffmpeg_vmaf_template": "test template",
				"report_file_name": "test_report.json"
			}`),
			want: Config{
				FfmpegPath:         NewConfigVal("test_ffmpeg"),
				FfprobePath:        NewConfigVal("test_ffprobe"),
				FfmpegVMAFTemplate: NewConfigVal("test template"),
				ReportFileName:     NewConfigVal("test_report.json"),
			},
		},
		"Partial": {
			given: []byte(`{
				"ffmpeg_path": "test_ffmpeg",
				"ffmpeg_vmaf_template": "test template"
			}`),
			want: Config{
				FfmpegPath:         NewConfigVal("test_ffmpeg"),
				FfmpegVMAFTemplate: NewConfigVal("test template"),
			},
		},
		"Empty JSON": {
			given: []byte(`{}`),
			want:  Config{},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Create config file with given contents.
			confFile := path.Join(t.TempDir(), fmt.Sprintf("config.%s", "json"))
			err := os.WriteFile(confFile, tt.given, 0o600)
			it.Must(t, err == nil, "Unexpected error: got = %v, want = nil", err)

			// Load config and assert contents are as expected.
			got, err := loadConfigFromFile(confFile)
			it.Should(t, err == nil, "Should be no error loading configuration from file: got = %v", err)

			it.Should(t, reflect.DeepEqual(got, tt.want), "got = %v, want = %v", got, tt.want)
		})
	}
}

func Test_Config_OverrideFrom(t *testing.T) {
	fixBaseConf := func() Config {
		return Config{
			FfmpegPath:         NewConfigVal("base_ffmpeg"),
			FfprobePath:        NewConfigVal("base_ffprobe"),
			FfmpegVMAFTemplate: NewConfigVal("base template"),
			ReportFileName:     NewConfigVal("base_report.json"),
		}
	}

	tests := map[string]struct {
		want        Config
		overrideSrc Config
	}{
		"Full config overrides all fields": {
			overrideSrc: Config{
				FfmpegPath:         NewConfigVal("test_ffmpeg"),
				FfprobePath:        NewConfigVal("test_ffprobe"),
				FfmpegVMAFTemplate: NewConfigVal("test template"),
				ReportFileName:     NewConfigVal("test_report.json"),
			},
			want: Config{
				FfmpegPath:         NewConfigVal("test_ffmpeg"),
				FfprobePath:        NewConfigVal("test_ffprobe"),
				FfmpegVMAFTemplate: NewConfigVal("test template"),
				ReportFileName:     NewConfigVal("test_report.json"),
			},
		},
		"Partial config overrides partial fields": {
			overrideSrc: Config{
				FfmpegPath:         NewConfigVal("test_ffmpeg"),
				FfmpegVMAFTemplate: NewConfigVal("test template"),
			},
			want: Config{
				// Overridden fields.
				FfmpegPath:         NewConfigVal("test_ffmpeg"),
				FfmpegVMAFTemplate: NewConfigVal("test template"),
				// Unmodified fields.
				FfprobePath:    NewConfigVal("base_ffprobe"),
				ReportFileName: NewConfigVal("base_report.json"),
			},
		},
		"Empty config does not override any fields": {
			overrideSrc: Config{},
			want:        fixBaseConf(),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Create a base Config object. This is the Config that we shall attempt to
			// override.
			given := fixBaseConf()

			// Attempt to override config from overrideSrc.
			given.OverrideFrom(tt.overrideSrc)

			it.Should(t, reflect.DeepEqual(given, tt.want), "got = %v, want = %v", given, tt.want)
		})
	}
}

func Test_DumpConfApp_Run(t *testing.T) {
	// This is one option we try to make sure is in dumped config file.
	want := `"report_file_name": "test_report.json"`

	// Create config file with given contents.
	configRaw := []byte("{" + want + "}")
	confFile := path.Join(t.TempDir(), fmt.Sprintf("config.%s", "json"))
	err := os.WriteFile(confFile, configRaw, 0o600)
	it.Must(t, err == nil, "Unexpected error: got = %v, want = nil", err)

	// Run command will generate encoding artifacts and analysis artifacts.
	cmd := CreateDumpConfCommand()

	// Redirect output to buffer
	commandOutput := &bytes.Buffer{}
	cmd.out = commandOutput

	err = cmd.Run([]string{"-conf", confFile})
	it.Should(t, err == nil, "Unexpected error running encode: got = %v, want = nil", err)
	// Check that config dump contains options we specified in config file.
	got := commandOutput.String()
	it.Should(t, strings.Contains(got, want), "got = %v, want to contain %v", got, want)
}

func Test_DumpConfApp_Run_WithNotFound(t *testing.T) {
	// This will make ffmpeg and ffprobe auto-detect to fail.
	t.Setenv("PATH", "")

	// Run command will generate encoding artifacts and analysis artifacts.
	cmd := CreateDumpConfCommand()

	// Redirect output to buffer
	commandOutput := &bytes.Buffer{}
	cmd.out = commandOutput

	err := cmd.Run([]string{})
	got := commandOutput.String()
	wantMatches := []string{
		`"ffmpeg_path": "not found"`,
		`"ffprobe_path": "not found"`,
	}
	for _, want := range wantMatches {
		it.Should(t, strings.Contains(got, want), "got = %v, want to contain %v", got, want)
	}

	wantErrMsg := "configuration validation"
	it.Must(t, err != nil, "Should have error")
	it.Should(t, strings.Contains(err.Error(), wantErrMsg), "got = %v, want to contain %v", err, wantErrMsg)
}

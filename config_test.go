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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_loadDefaultConfig(t *testing.T) {
	c, err := loadDefaultConfig()
	assert.NoError(t, err, "Should create DefaultConfig without errors")

	assert.NoError(t, c.Verify(), "DefaultConfig should be valid")
}

func Test_loadDefaultConfig_Negative(t *testing.T) {
	// Messing up PATH should result in failure detecting ffmpeg and ffprobe which
	// should result in error from calling DefaultConfig().
	t.Setenv("PATH", "")
	_, err := loadDefaultConfig()
	assert.ErrorContains(t, err, "DefaultConfig: ")
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
				"libvmaf_model_path": "test_libvmaf_model.json",
				"ffmpeg_vmaf_template": "test template",
				"report_file_name": "test_report.json"
			}`),
			want: Config{
				FfmpegPath:         NewConfigVal("test_ffmpeg"),
				FfprobePath:        NewConfigVal("test_ffprobe"),
				LibvmafModelPath:   NewConfigVal("test_libvmaf_model.json"),
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
			require.NoError(t, err)

			// Load config and assert contents are as expected.
			got, err := loadConfigFromFile(confFile)
			assert.NoError(t, err, "Should be no error loading configuration from file")

			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_Config_OverrideFrom(t *testing.T) {
	fixBaseConf := func() Config {
		return Config{
			FfmpegPath:         NewConfigVal("base_ffmpeg"),
			FfprobePath:        NewConfigVal("base_ffprobe"),
			LibvmafModelPath:   NewConfigVal("base_libvmaf_model.json"),
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
				LibvmafModelPath:   NewConfigVal("test_libvmaf_model.json"),
				FfmpegVMAFTemplate: NewConfigVal("test template"),
				ReportFileName:     NewConfigVal("test_report.json"),
			},
			want: Config{
				FfmpegPath:         NewConfigVal("test_ffmpeg"),
				FfprobePath:        NewConfigVal("test_ffprobe"),
				LibvmafModelPath:   NewConfigVal("test_libvmaf_model.json"),
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
				FfprobePath:      NewConfigVal("base_ffprobe"),
				LibvmafModelPath: NewConfigVal("base_libvmaf_model.json"),
				ReportFileName:   NewConfigVal("base_report.json"),
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

			assert.Equal(t, tt.want, given)
		})
	}
}

func Test_DumpConfApp_Run(t *testing.T) {
	commandOutput := &bytes.Buffer{}
	// This is one option we try to make sure is in dumped config file.
	want := `"report_file_name": "test_report.json"`

	// Create config file with given contents.
	configRaw := []byte("{" + want + "}")
	confFile := path.Join(t.TempDir(), fmt.Sprintf("config.%s", "json"))
	require.NoError(t, os.WriteFile(confFile, configRaw, 0o600))

	// Run command will generate encoding artifacts and analysis artifacts.
	cmd := CreateDumpConfCommand()

	// Redirect output to buffer
	cmd.out = commandOutput

	err := cmd.Run([]string{"-conf", confFile})
	assert.NoError(t, err, "Unexpected error running encode")
	// Check that config dump contains options we specified in config file.
	assert.Contains(t, commandOutput.String(), want)
}

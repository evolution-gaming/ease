// Copyright ©2022 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Tests for ease tool subcommands.
package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/evolution-gaming/ease/internal/encoding"
	"github.com/evolution-gaming/ease/internal/it"
	"github.com/evolution-gaming/ease/internal/testutil"
)

// Happy path functional test for run sub-command.
func Test_RunApp_Run(t *testing.T) {
	testutil.EnsureFfmpegWithVMAF(t)
	tempDir := t.TempDir()
	ePlan := fixPlanConfig(t)
	outDir := path.Join(tempDir, "out")

	t.Run("Should succeed execution with -plan flag", func(t *testing.T) {
		// Run command will generate encoding artifacts and analysis artifacts.
		app := CreateRunCommand()
		err := app.Run([]string{"-plan", ePlan, "-out-dir", outDir})
		it.Should(t, err == nil, "Unexpected error running encode: got = %v, want = nil", err)
	})

	t.Run("Should have a CSV report file", func(t *testing.T) {
		fd, err2 := os.Open(path.Join(outDir, "report.csv"))
		it.Must(t, err2 == nil, "Unexpected error opening report.csv: got = %v, want = nil", err2)
		defer fd.Close()
		records, err3 := csv.NewReader(fd).ReadAll()
		it.Must(t, err3 == nil, "Unexpected error reading CSV records: got = %v, want = nil", err3)
		// Expect 2 records: CSV header + record for 1 encoding.
		wantCount := 2
		gotCount := len(records)
		it.Should(t, gotCount == wantCount, "Unexpected number of records in report file: got = %v, want = %v", gotCount, wantCount)
	})

	t.Run("Should create plots", func(t *testing.T) {
		plotFileGlobs := []string{
			fmt.Sprintf("%s/*/*bitrate.png", outDir),
			fmt.Sprintf("%s/*/*vmaf.png", outDir),
			fmt.Sprintf("%s/*/*psnr.png", outDir),
		}

		for _, g := range plotFileGlobs {
			plots, _ := filepath.Glob(g)
			it.Must(t, plots != nil, "Glob match should not be nil")
			got := len(plots)
			it.Should(t, got == 1, "Expecting one file matching glob \"%v\": got = %v", g, got)
		}
	})
}

/*************************************
* Negative tests for run sub-command.
 *************************************/

// Error cases for run sub-command flags.
func Test_RunApp_Run_FlagErrors(t *testing.T) {
	// For some cases we need existing plan config file.
	planConfig := fixPlanConfig(t)

	tempDir := t.TempDir()

	tests := map[string]struct {
		// substring in Error()
		want      string
		givenArgs []string
	}{
		"Wrong flags": {
			givenArgs: []string{"-zzz", "aaaa", "-plan", planConfig, "-out-dir", path.Join(tempDir, "out1")},
			want:      "run usage error",
		},
		"Mandatory plan flag missing": {
			givenArgs: []string{"-out-dir", path.Join(tempDir, "out2")},
			want:      "mandatory option -plan is missing",
		},
		"Mandatory out-dir flag missing": {
			givenArgs: []string{"-plan", planConfig},
			want:      "mandatory option -out-dir is missing",
		},
		"Non-existent plan": {
			givenArgs: []string{"-plan", "a/yyy", "-out-dir", path.Join(tempDir, "out3")},
			want:      "encoding plan file does not exist?",
		},
		"Non-existent config file": {
			givenArgs: []string{"-conf", "missing-conf.json", "-plan", planConfig, "-out-dir", path.Join(tempDir, "out4v")},
			want:      "no such file or directory",
		},
		"Empty flags": {
			givenArgs: []string{},
			want:      "mandatory option",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			cmd := CreateRunCommand()
			// Discard usage output so that during test execution test output is
			// not flooded with command Usage/Help stuff.
			cmd.fs.SetOutput(io.Discard)
			gotErr := cmd.Run(tc.givenArgs)
			it.Must(t, gotErr != nil, "Should have error")
			it.Should(t, strings.Contains(gotErr.Error(), tc.want), "got = %v, want to contain %v", gotErr, tc.want)
		})
	}
}

func Test_RunApp_Run_WithFailedVQM(t *testing.T) {
	// Create a fake ffmpeg and modify PATH so that it's picked up first and
	// blows up VQM calculation.
	fixCreateFakeFfmpegAndPutItOnPath(t)

	app := CreateRunCommand()
	plan := fixPlanConfig(t)
	outDir := path.Join(t.TempDir(), "out")

	wantErrMsg := "FFmpeg VMAF support validation"
	wantExitCode := 1
	gotErr := app.Run([]string{"-plan", plan, "-out-dir", outDir})
	it.Must(t, gotErr != nil, "Should have error")
	it.Should(t, strings.Contains(gotErr.Error(), wantErrMsg), "got = %v, want to contain %v", gotErr, wantErrMsg)

	var appErr *AppError
	it.Must(t, errors.As(gotErr, &appErr), "Expecting error of type AppError: got = %v", gotErr)

	gotExitCode := appErr.ExitCode()
	it.Should(t, gotExitCode == wantExitCode, "Exit code mismatch: got = %v, want = %v", gotExitCode, wantExitCode)
}

func Test_RunApp_Run_WithInvalidPlanConfigParseError(t *testing.T) {
	app := CreateRunCommand()
	wantErrMsg := "PlanConfig not valid: validation error with reasons"
	wantExitCode := 1

	gotErr := app.Run([]string{"-plan", fixPlanConfigInvalid(t), "-out-dir", t.TempDir()})
	it.Must(t, gotErr != nil, "Should have error")
	it.Should(t, strings.Contains(gotErr.Error(), wantErrMsg), "got = %v, want to contain %v", gotErr, wantErrMsg)

	var appErr *AppError
	it.Must(t, errors.As(gotErr, &appErr), "Expecting error of type AppError: got = %v", gotErr)

	gotExitCode := appErr.ExitCode()
	it.Should(t, gotExitCode == wantExitCode, "Exit code mismatch: got = %v, want = %v", gotExitCode, wantExitCode)
}

func Test_RunApp_Run_WithNonEmptyOutDirShouldTerminate(t *testing.T) {
	app := CreateRunCommand()
	plan := fixPlanConfig(t)
	// Dir containing plan file by definition is non-empty.
	outDir := path.Dir(plan)

	t.Logf("Given existing out dir: %s", outDir)
	err := os.MkdirAll(outDir, 0o755)
	it.Must(t, err == nil, "Unexpected error: got = %v, want = nil", err)

	t.Log("When plan is executed")
	gotErr := app.Run([]string{"-plan", plan, "-out-dir", outDir})

	t.Log("Then there is an error and program terminates")
	wantErrMsg := "non-empty out dir"
	it.Must(t, gotErr != nil, "Should have error")
	it.Should(t, strings.Contains(gotErr.Error(), wantErrMsg), "got = %v, want to contain %v", gotErr, wantErrMsg)

	var appErr *AppError
	it.Must(t, errors.As(gotErr, &appErr), "Expecting error of type AppError: got = %v", gotErr)

	wantExitCode := 1
	gotExitCode := appErr.ExitCode()
	it.Should(t, gotExitCode == wantExitCode, "Exit code mismatch: got = %v, want = %v", gotExitCode, wantExitCode)
}

func Test_RunApp_Run_WithInvalidApplicationConfig(t *testing.T) {
	invalidConfig := []byte(`{}`)
	confFile := path.Join(t.TempDir(), "wrong.json")
	// Empty configuration is wrong configuration. When we explicitly specify
	// configuration file, we expect all options to be defined.
	err := os.WriteFile(confFile, invalidConfig, 0o600)
	it.Must(t, err == nil, "Unexpected error: got = %v, want = nil", err)

	app := CreateRunCommand()
	gotErr := app.Run([]string{"-plan", fixPlanConfigInvalid(t), "-out-dir", t.TempDir(), "-conf", confFile})

	var expErr *AppError
	it.Must(t, errors.As(gotErr, &expErr), "Expecting error of type AppError: got = %v", gotErr)
}

func Test_RunApp_Run_MisalignedFrames(t *testing.T) {
	plan := fixPlanConfigMisalignedFrames(t)
	app := CreateRunCommand()
	gotErr := app.Run([]string{"-plan", plan, "-out-dir", t.TempDir()})

	var expErr *AppError
	it.Must(t, errors.As(gotErr, &expErr), "Expecting error of type AppError: got = %v", gotErr)
	wantErrMsg := "VQM calculations had errors, see log for reasons"
	it.Should(t, strings.Contains(gotErr.Error(), wantErrMsg), "got = %v, want to contain %v", gotErr, wantErrMsg)
}

// Functional tests for other sub-commands..
func TestIntegration_AllSubcommands(t *testing.T) {
	testutil.EnsureFfmpegWithVMAF(t)
	tempDir := t.TempDir()
	outDir := path.Join(tempDir, "out")
	ePlan := fixPlanConfig(t)

	// Run command will generate encoding artifacts and analysis artifacts for later use
	// ans inputs.
	err := CreateRunCommand().Run([]string{"-plan", ePlan, "-out-dir", outDir})
	it.Must(t, err == nil, "Unexpected error: got = %v, want = nil", err)

	t.Run("vqmplot should create plots", func(t *testing.T) {
		var vqmFile string
		// Need to get file with VQMs from encode stage.
		m, _ := filepath.Glob(fmt.Sprintf("%s/*vqm.json", outDir))
		wantMatches := 1
		gotMatches := len(m)
		it.Must(t, gotMatches == wantMatches, "got = %v, want = %v", gotMatches, wantMatches)
		vqmFile = m[0]

		for _, metric := range []string{"VMAF", "PSNR", "MS-SSIM"} {
			t.Run(metric, func(t *testing.T) {
				outFile := path.Join(tempDir, fmt.Sprintf("vqmplot_%s.png", metric))
				err := CreateVQMPlotCommand().Run([]string{"-i", vqmFile, "-o", outFile, "-m", metric, "-fps", "24"})
				it.Should(t, err == nil, "Unexpected error running vqmplot: got = %v, want = nil", err)
				it.Should(t, testutil.FileExists(outFile), "VQM file missing: %v", outFile)
			})
		}
	})

	t.Run("bitrate should create bitrate plot", func(t *testing.T) {
		var compressedFile string
		// Need to get compressed file from encode stage.
		m, _ := filepath.Glob(fmt.Sprintf("%s/*.mp4", outDir))
		wantMatches := 1
		gotMatches := len(m)
		it.Must(t, gotMatches == wantMatches, "got = %v, want = %v", gotMatches, wantMatches)
		compressedFile = m[0]

		outFile := path.Join(tempDir, "bitrate.png")
		err := CreateBitrateCommand().Run([]string{"-i", compressedFile, "-o", outFile})
		it.Should(t, err == nil, "Unexpected error running bitrate: got = %v, want = nil", err)
		it.Should(t, testutil.FileExists(outFile), "bitrate plot file missing: %v", outFile)
	})

	t.Run("new-plan should create plan template", func(t *testing.T) {
		planFile := path.Join(t.TempDir(), "plan.json")
		err := CreateNewPlanCommand().Run([]string{"-i", "video1.mp4", "-o", planFile})
		it.Must(t, err == nil, "Unexpected error: got = %v, want = nil", err)

		b, err := os.ReadFile(planFile)
		it.Must(t, err == nil, "Unexpected error: got = %v, want = nil", err)
		pc, err := encoding.NewPlanConfigFromJSON(b)
		it.Must(t, err == nil, "Unexpected error: got = %v, want = nil", err)

		it.Must(t, len(pc.Inputs) == 1, "got = %v, want = 1", len(pc.Inputs))
		it.Should(t, pc.Inputs[0] == "video1.mp4", "got = %v, want = %v", pc.Inputs[0], "video1.mp4")
		it.Should(t, len(pc.Schemes) > 0, "got = %v, want > 0", len(pc.Schemes))
	})
}

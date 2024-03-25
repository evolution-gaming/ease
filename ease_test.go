// Copyright ©2022 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Tests for ease tool subcommands.
package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/evolution-gaming/ease/internal/encoding"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Happy path functional test for run sub-command.
func Test_RunApp_Run(t *testing.T) {
	tempDir := t.TempDir()
	ePlan := fixPlanConfig(t)
	outDir := path.Join(tempDir, "out")

	t.Run("Should succeed execution with -plan flag", func(t *testing.T) {
		// Run command will generate encoding artifacts and analysis artifacts.
		app := CreateRunCommand()
		err := app.Run([]string{"-plan", ePlan, "-out-dir", outDir})
		assert.NoError(t, err, "Unexpected error running encode")
	})

	t.Run("Should have a CSV report file", func(t *testing.T) {
		fd, err2 := os.Open(path.Join(outDir, "report.csv"))
		assert.NoError(t, err2, "Unexpected error opening report.csv")
		defer fd.Close()
		records, err3 := csv.NewReader(fd).ReadAll()
		assert.NoError(t, err3, "Unexpected error reading CSV records")
		// Expect 2 records: CSV header + record for 1 encoding.
		assert.Len(t, records, 2, "Unexpected number of records in report file")
	})

	t.Run("Should create plots", func(t *testing.T) {
		bitratePlots, _ := filepath.Glob(fmt.Sprintf("%s/*/*bitrate.png", outDir))
		assert.Len(t, bitratePlots, 1, "Expecting one file for bitrate plot")

		vmafPlots, _ := filepath.Glob(fmt.Sprintf("%s/*/*vmaf.png", outDir))
		assert.Len(t, vmafPlots, 1, "Expecting one file for VMAF plot")

		psnrPlots, _ := filepath.Glob(fmt.Sprintf("%s/*/*psnr.png", outDir))
		assert.Len(t, psnrPlots, 1, "Expecting one file for PSNR plot")
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
			assert.ErrorContains(t, gotErr, tc.want)
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

	wantErrMsg := "VQM calculations had errors, see log for reasons"
	wantExitCode := 1
	gotErr := app.Run([]string{"-plan", plan, "-out-dir", outDir})
	assert.ErrorContains(t, gotErr, wantErrMsg)

	gotExitCode := gotErr.(*AppError).ExitCode()
	assert.Equal(t, wantExitCode, gotExitCode, "Exit code mismatch")
}

func Test_RunApp_Run_WithInvalidPlanConfigParseError(t *testing.T) {
	app := CreateRunCommand()
	wantErrMsg := "PlanConfig not valid: validation error with reasons"
	wantExitCode := 1

	gotErr := app.Run([]string{"-plan", fixPlanConfigInvalid(t), "-out-dir", t.TempDir()})
	assert.ErrorContains(t, gotErr, wantErrMsg)

	gotExitCode := gotErr.(*AppError).ExitCode()
	assert.Equal(t, wantExitCode, gotExitCode, "Exit code mismatch")
}

func Test_RunApp_Run_WithNonEmptyOutDirShouldTerminate(t *testing.T) {
	app := CreateRunCommand()
	plan := fixPlanConfig(t)
	// Dir containing plan file by definition is non-empty.
	outDir := path.Dir(plan)

	t.Logf("Given existing out dir: %s", outDir)
	require.NoError(t, os.MkdirAll(outDir, 0o755))

	t.Log("When plan is executed")
	gotErr := app.Run([]string{"-plan", plan, "-out-dir", outDir})

	t.Log("Then there is an error and program terminates")
	wantErrMsg := "non-empty out dir"
	assert.ErrorContains(t, gotErr, wantErrMsg)

	wantExitCode := 1
	gotExitCode := gotErr.(*AppError).ExitCode()
	assert.Equal(t, wantExitCode, gotExitCode, "Exit code mismatch")
}

func Test_RunApp_Run_WithInvalidApplicationConfig(t *testing.T) {
	invalidConfig := []byte(`{}`)
	confFile := path.Join(t.TempDir(), "wrong.json")
	// Empty configuration is wrong configuration. When we explicitly specify
	// configuration file, we expect all options to be defined.
	err := os.WriteFile(confFile, invalidConfig, 0o600)
	require.NoError(t, err)

	app := CreateRunCommand()
	gotErr := app.Run([]string{"-plan", fixPlanConfigInvalid(t), "-out-dir", t.TempDir(), "-conf", confFile})

	var expErr *AppError
	assert.ErrorAs(t, gotErr, &expErr, "Expecting error of type AppError")
}

func Test_RunApp_Run_MisalignedFrames(t *testing.T) {
	plan := fixPlanConfigMisalignedFrames(t)
	app := CreateRunCommand()
	gotErr := app.Run([]string{"-plan", plan, "-out-dir", t.TempDir()})

	var expErr *AppError
	assert.ErrorAs(t, gotErr, &expErr, "Expecting error of type AppError")
	assert.ErrorContains(t, gotErr, "VQM calculations had errors, see log for reasons")
}

// Functional tests for other sub-commands..
func TestIntegration_AllSubcommands(t *testing.T) {
	tempDir := t.TempDir()
	outDir := path.Join(tempDir, "out")
	ePlan := fixPlanConfig(t)

	// Run command will generate encoding artifacts and analysis artifacts for later use
	// ans inputs.
	err := CreateRunCommand().Run([]string{"-plan", ePlan, "-out-dir", outDir})
	require.NoError(t, err)

	t.Run("vqmplot should create plots", func(t *testing.T) {
		var vqmFile string
		// Need to get file with VQMs from encode stage.
		m, _ := filepath.Glob(fmt.Sprintf("%s/*vqm.json", outDir))
		assert.Len(t, m, 1)
		vqmFile = m[0]

		for _, metric := range []string{"VMAF", "PSNR", "MS-SSIM"} {
			t.Run(metric, func(t *testing.T) {
				outFile := path.Join(tempDir, fmt.Sprintf("vqmplot_%s.png", metric))
				err := CreateVQMPlotCommand().Run([]string{"-i", vqmFile, "-o", outFile, "-m", metric})
				assert.NoError(t, err, "Unexpected error running vqmplot")
				assert.FileExists(t, outFile, "VQM file missing")
			})
		}
	})

	t.Run("bitrate should create bitrate plot", func(t *testing.T) {
		var compressedFile string
		// Need to get compressed file from encode stage.
		m, _ := filepath.Glob(fmt.Sprintf("%s/*.mp4", outDir))
		assert.Len(t, m, 1)
		compressedFile = m[0]

		outFile := path.Join(tempDir, "bitrate.png")
		err := CreateBitrateCommand().Run([]string{"-i", compressedFile, "-o", outFile})
		assert.NoError(t, err, "Unexpected error running bitrate")
		assert.FileExists(t, outFile, "bitrate plot file missing")
	})

	t.Run("new-plan should create plan template", func(t *testing.T) {
		planFile := path.Join(t.TempDir(), "plan.json")
		err := CreateNewPlanCommand().Run([]string{"-i", "video1.mp4", "-o", planFile})
		assert.NoError(t, err)

		b, err := os.ReadFile(planFile)
		assert.NoError(t, err)
		pc, err := encoding.NewPlanConfigFromJSON(b)
		assert.NoError(t, err)

		assert.Len(t, pc.Inputs, 1)
		assert.Equal(t, pc.Inputs[0], "video1.mp4")
		assert.True(t, len(pc.Schemes) > 0)
	})
}

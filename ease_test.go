// Copyright ©2022 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Tests for ease tool subcommands.
package main

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// Happy path functional test for run sub-command.
func TestRunApp_Run(t *testing.T) {
	tempDir := t.TempDir()
	ePlan := fixPlanConfig(t)
	outDir := path.Join(tempDir, "out")

	t.Log("Should succeed execution with -plan flag")
	// Run command will generate encoding artifacts and analysis artifacts.
	err := CreateRunCommand().Run([]string{"-plan", ePlan, "-out-dir", outDir})
	if err != nil {
		t.Errorf("Unexpected error running encode: %v", err)
	}

	buf, err2 := os.ReadFile(path.Join(outDir, "report.json"))
	if err2 != nil {
		t.Errorf("Unexpected error: %v", err2)
	}
	if len(buf) == 0 {
		t.Errorf("No data in report file")
	}

	t.Log("Analyse should create bitrate, VMAF, PSNR and SSIM plots")
	if m, _ := filepath.Glob(fmt.Sprintf("%s/*/*bitrate.png", outDir)); len(m) != 1 {
		t.Errorf("Expecting one file for bitrate plot, got: %s", m)
	}
	if m, _ := filepath.Glob(fmt.Sprintf("%s/*/*vmaf.png", outDir)); len(m) != 1 {
		t.Errorf("Expecting one file for VMAF plot, got: %s", m)
	}
	if m, _ := filepath.Glob(fmt.Sprintf("%s/*/*psnr.png", outDir)); len(m) != 1 {
		t.Errorf("Expecting one file for PSNR plot, got: %s", m)
	}
	if m, _ := filepath.Glob(fmt.Sprintf("%s/*/*ms-ssim.png", outDir)); len(m) != 1 {
		t.Errorf("Expecting one file for MS-SSIM plot, got: %s", m)
	}
}

// Error cases for run sub-command flags.
func TestRunApp_FlagErrors(t *testing.T) {
	tests := map[string]struct {
		// substring in Error()
		want      string
		givenArgs []string
	}{
		"Wrong flags": {
			givenArgs: []string{"-zzz", "aaaa", "-plan", "a/xxx", "-out-dir", "out"},
			want:      "run usage error",
		},
		"Mandatory plan flag missing": {
			givenArgs: []string{"-out-dir", "out"},
			want:      "mandatory option -plan is missing",
		},
		"Mandatory out-dir flag missing": {
			givenArgs: []string{"-plan", "a/xxx"},
			want:      "mandatory option -out-dir is missing",
		},
		"Non-existent conf": {
			givenArgs: []string{"-plan", "a/yyy", "-out-dir", "out"},
			want:      "encoding plan file does not exist?",
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
			if c, ok := cmd.(*App); ok {
				c.fs.SetOutput(io.Discard)
			}
			gotErr := cmd.Run(tc.givenArgs)
			if !strings.Contains(gotErr.Error(), tc.want) {
				t.Errorf("Error mismatch (-want +got):\n-%s\n+%s\n", tc.want, gotErr.Error())
			}
		})
	}
}

/*************************************
* Negative tests for run sub-command.
 *************************************/

func TestRunApp_Run_WithFailedVQM(t *testing.T) {
	// Create a fake ffmpeg and modify PATH so that it's picked up first and
	// blows up VQM calculation.
	fixCreateFakeFfmpegAndPutItOnPath(t)

	app := CreateRunCommand()
	plan := fixPlanConfig(t)
	outDir := path.Join(t.TempDir(), "out")

	gotErr := app.Run([]string{"-plan", plan, "-out-dir", outDir})
	if gotErr == nil {
		t.Fatal("Error expected but go <nil>")
	}
	wantErrMsg := "VQM calculations had errors, see log for reasons"
	wantExitCode := 1

	if diff := cmp.Diff(wantErrMsg, gotErr.Error()); diff != "" {
		t.Errorf("Error message mismatch (-want +got):\n%s", diff)
	}

	gotExitCode := gotErr.(*AppError).ExitCode()
	if diff := cmp.Diff(wantExitCode, gotExitCode); diff != "" {
		t.Errorf("ExitCode mismatch (-want +got):\n%s", diff)
	}
}

func TestRunApp_Run_WithInvalidPlanConfigParseError(t *testing.T) {
	app := CreateRunCommand()
	gotErr := app.Run([]string{"-plan", fixPlanConfigInvalid(t), "-out-dir", t.TempDir()})
	if gotErr == nil {
		t.Fatal("Error expected but go <nil>")
	}
	wantErrMsg := "PlanConfig not valid: validation error with reasons"
	wantExitCode := 1

	if !strings.HasPrefix(gotErr.Error(), wantErrMsg) {
		t.Errorf("Error message mismatch (-want +got):\n-%s\n+%s", wantErrMsg, gotErr.Error())
	}

	gotExitCode := gotErr.(*AppError).ExitCode()
	if diff := cmp.Diff(wantExitCode, gotExitCode); diff != "" {
		t.Errorf("ExitCode mismatch (-want +got):\n%s", diff)
	}
}

func TestRunApp_Run_WithNonEmptyOutDirShouldTerminate(t *testing.T) {
	app := CreateRunCommand()
	plan := fixPlanConfig(t)
	// Dir containing plan file by definition is non-empty.
	outDir := path.Dir(plan)

	t.Logf("Given existing out dir: %s", outDir)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatalf("Cannot create out dir: %s", err)
	}

	t.Log("When plan is executed")
	gotErr := app.Run([]string{"-plan", plan, "-out-dir", outDir})

	t.Log("Then there is an error and program terminates")
	if gotErr == nil {
		t.Fatal("Error expected but go <nil>")
	}
	wantErrMsg := "non-empty out dir"
	wantExitCode := 1

	if !strings.HasPrefix(gotErr.Error(), wantErrMsg) {
		t.Errorf("Error message mismatch (-want +got):\n-%s\n+%s", wantErrMsg, gotErr.Error())
	}

	gotExitCode := gotErr.(*AppError).ExitCode()
	if diff := cmp.Diff(wantExitCode, gotExitCode); diff != "" {
		t.Errorf("ExitCode mismatch (-want +got):\n%s", diff)
	}
}

// Functional tests for other sub-commands..
func TestIntegration_AllSubcommands(t *testing.T) {
	tempDir := t.TempDir()
	outDir := path.Join(tempDir, "out")
	ePlan := fixPlanConfig(t)

	// Run command will generate encoding artifacts and analysis artifacts for later use
	// ans inputs.
	err := CreateRunCommand().Run([]string{"-plan", ePlan, "-out-dir", outDir})
	if err != nil {
		t.Fatalf("Unexpected during plan execution: %v", err)
	}

	t.Run("Vqmplot should create plots", func(t *testing.T) {
		var vqmFile string
		// Need to get file with VQMs from encode stage.
		if m, _ := filepath.Glob(fmt.Sprintf("%s/*vqm.json", outDir)); len(m) != 1 {
			t.Errorf("Expecting one file with VQM data, got: %s", m)
		} else {
			vqmFile = m[0]
		}

		for _, metric := range []string{"VMAF", "PSNR", "MS-SSIM"} {
			t.Run(metric, func(t *testing.T) {
				outFile := path.Join(tempDir, fmt.Sprintf("vqmplot_%s.png", metric))
				err := CreateVQMPlotCommand().Run([]string{"-i", vqmFile, "-o", outFile, "-m", metric})
				if err != nil {
					t.Errorf("Unexpected error running vqmplot: %v", err)
				}
				if _, err := os.Stat(outFile); os.IsNotExist(err) {
					t.Errorf("VQM file missing: %s", outFile)
				}
			})
		}
	})

	t.Run("Bitrate should create bitrate plot", func(t *testing.T) {
		var compressedFile string
		// Need to get compressed file from encode stage.
		if m, _ := filepath.Glob(fmt.Sprintf("%s/*.mp4", outDir)); len(m) != 1 {
			t.Errorf("Expecting one compressed file, got: %s", m)
		} else {
			compressedFile = m[0]
		}

		outFile := path.Join(tempDir, "bitrate.png")
		err := CreateBitrateCommand().Run([]string{"-i", compressedFile, "-o", outFile})
		if err != nil {
			t.Errorf("Unexpected error running bitrate: %v", err)
		}
		if _, err := os.Stat(outFile); os.IsNotExist(err) {
			t.Errorf("bitrate plot file missing: %s", outFile)
		}
	})
}

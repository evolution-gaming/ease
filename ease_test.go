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

// Encode subcommand related tests.
func TestEncodeApp_WrongFlags(t *testing.T) {
	tests := map[string]struct {
		// substring in Error()
		want      string
		givenArgs []string
	}{
		"Wrong flags": {
			givenArgs: []string{"-zzz", "aaaa", "-plan", "a/xxx"},
			want:      "encode usage error",
		},
		"Empty flags": {
			givenArgs: []string{""},
			want:      "mandatory option -plan is missing",
		},
		"Non-existent conf": {
			givenArgs: []string{"-plan", "a/yyy"},
			want:      "encoding plan file does not exist?",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			cmd := CreateEncodeCommand()
			// Discard usage output so that during test execution test output is
			// not flooded with command Usage/Help stuff.
			if c, ok := cmd.(*EncodeApp); ok {
				c.fs.SetOutput(io.Discard)
			}
			gotErr := cmd.Run(tc.givenArgs)
			if !strings.Contains(gotErr.Error(), tc.want) {
				t.Errorf("Error mismatch (-want +got):\n-%s\n+%s\n", tc.want, gotErr.Error())
			}
		})
	}
}

func TestEncodeApp_Run_WithFailedVQM(t *testing.T) {
	// Create a fake ffmpeg and modify PATH so that it's picked up first and
	// blows up VQM calculation.
	fixCreateFakeFfmpegAndPutItOnPath(t)

	app := CreateEncodeCommand()
	plan, _ := fixPlanConfig(t)
	gotErr := app.Run([]string{"-plan", plan})
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

func TestEncodeApp_Run_WithInvalidPlanConfigParseError(t *testing.T) {
	app := CreateEncodeCommand()
	gotErr := app.Run([]string{"-plan", fixPlanConfigInvalid(t)})
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

func TestEncodeApp_Run(t *testing.T) {
	plan, _ := fixPlanConfig(t)
	t.Run("Should succeed execution with -plan flag", func(t *testing.T) {
		// Since we do not specify -report option, report content will end up on
		// stdout, we want to redirect stdout to avoid flooding test output and
		// also to do minimal checks.
		redirStdout := path.Join(t.TempDir(), "report.json")
		redirectStdout(redirStdout, t)
		app := CreateEncodeCommand()
		err := app.Run([]string{"-plan", plan})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		buf, err2 := os.ReadFile(redirStdout)
		if err2 != nil {
			t.Errorf("Unexpected error: %v", err2)
		}
		if len(buf) == 0 {
			t.Errorf("No data in report file")
		}
	})
	t.Run("Should succeed execution with -report flag", func(t *testing.T) {
		app := CreateEncodeCommand()
		report := path.Join(t.TempDir(), "report.json")
		err := app.Run([]string{"-plan", plan, "-report", report})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		buf, err2 := os.ReadFile(report)
		if err2 != nil {
			t.Errorf("Unexpected error: %v", err2)
		}
		if len(buf) == 0 {
			t.Errorf("No data in report file")
		}
	})
	t.Run("Should succeed execution with -dry-run flag", func(t *testing.T) {
		app := CreateEncodeCommand()
		err := app.Run([]string{"-dry-run", "-plan", plan})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}

// Analyse subcommand related tests.
func TestAnalyseApp_WrongFlags(t *testing.T) {
	tests := map[string]struct {
		// substring in Error()
		want      string
		givenArgs []string
	}{
		"Wrong flags": {
			givenArgs: []string{"-zzz", "aaaa", "-version"},
			want:      "analyse usage error",
		},
		"Mandatory -report flag": {
			givenArgs: []string{"-out-dir", "/tmp"},
			want:      "mandatory option -report is missing",
		},

		"Mandatory -out-dir flag": {
			givenArgs: []string{"-report", "testdata/encoding_artifacts/report.json"},
			want:      "mandatory option -out-dir is missing",
		},
		"Non-existent conf": {
			givenArgs: []string{"-report", "a/yyy", "-out-dir", "/tmp"},
			want:      "report file does not exist?",
		},
	}

	for name, tc := range tests {
		wantExitCode := 2
		t.Run(name, func(t *testing.T) {
			cmd := CreateAnalyseCommand()
			// Discard usage output so that during test execution test output is
			// not flooded with command Usage/Help stuff.
			if c, ok := cmd.(*AnalyseApp); ok {
				c.fs.SetOutput(io.Discard)
			}
			gotErr := cmd.Run(tc.givenArgs)
			if !strings.Contains(gotErr.Error(), tc.want) {
				t.Errorf("Error mismatch (-want +got):\n-%s\n+%s\n", tc.want, gotErr.Error())
			}
			if e, ok := gotErr.(*AppError); ok {
				gotExitCode := e.ExitCode()
				if diff := cmp.Diff(wantExitCode, gotExitCode); diff != "" {
					t.Errorf("ExitCode mismatch (-want +got):\n%s", diff)
				}
			} else {
				t.Errorf("Unexpected error type: %v", gotErr)
			}
		})
	}
}

// Integration tests for ease tool.
func TestIntegration_AllSubcommands(t *testing.T) {
	tempDir := t.TempDir()
	ePlan, encOutDir := fixPlanConfig(t)
	report := path.Join(tempDir, "report.json")
	analyseOutDir := path.Join(tempDir, "out")

	// Encode command will generate artifacts needed for other subcommands, so
	// it is more like precondition.
	err := CreateEncodeCommand().Run([]string{"-plan", ePlan, "-report", report})
	if err != nil {
		t.Errorf("Unexpected error running encode: %v", err)
	}

	t.Run("Analyse should create bitrate, VMAF, PSNR and SSIM plots", func(t *testing.T) {
		// Run analyse subcommand.
		err := CreateAnalyseCommand().Run([]string{"-report", report, "-out-dir", analyseOutDir})
		if err != nil {
			t.Errorf("Unexpected error running analysis: %v", err)
		}

		if m, _ := filepath.Glob(fmt.Sprintf("%s/*/*bitrate.png", analyseOutDir)); len(m) != 1 {
			t.Errorf("Expecting one file for bitrate plot, got: %s", m)
		}
		if m, _ := filepath.Glob(fmt.Sprintf("%s/*/*vmaf.png", analyseOutDir)); len(m) != 1 {
			t.Errorf("Expecting one file for VMAF plot, got: %s", m)
		}
		if m, _ := filepath.Glob(fmt.Sprintf("%s/*/*psnr.png", analyseOutDir)); len(m) != 1 {
			t.Errorf("Expecting one file for PSNR plot, got: %s", m)
		}
		if m, _ := filepath.Glob(fmt.Sprintf("%s/*/*ms-ssim.png", analyseOutDir)); len(m) != 1 {
			t.Errorf("Expecting one file for MS-SSIM plot, got: %s", m)
		}
	})

	t.Run("Vqmplot should create plots", func(t *testing.T) {
		var vqmFile string
		// Need to get file with VQMs from encode stage.
		if m, _ := filepath.Glob(fmt.Sprintf("%s/*vqm.json", encOutDir)); len(m) != 1 {
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
		if m, _ := filepath.Glob(fmt.Sprintf("%s/*.mp4", encOutDir)); len(m) != 1 {
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

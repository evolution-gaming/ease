// Copyright ©2022 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Encoding plan related tests.

package encoding

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/evolution-gaming/ease/internal/it"
	"github.com/evolution-gaming/ease/internal/testutil"
)

func TestCreatePlanFromConfig(t *testing.T) {
	// Given plan configuration
	planConfig := PlanConfig{
		Inputs: []string{"videos/clip01.mp4", "videos/clip02.mp4"},
		Schemes: []Scheme{
			{"x264 param1 x", "ffmpeg -i %INPUT% -param1 x -y %OUTPUT%.mp4"},
			{"x264_param1_y", "ffmpeg -i %INPUT% -param1 y -y %OUTPUT%.mp4"},
		},
	}
	// When I create a new Plan from PlanConfig
	plan := NewPlan(planConfig, "out")
	var gotCommands, gotOutputFiles []string
	for _, c := range plan.Commands {
		gotCommands = append(gotCommands, c.Cmd)
		gotOutputFiles = append(gotOutputFiles, c.OutputFile)
	}

	t.Run("Should have expected encoding commands", func(t *testing.T) {
		// Then I get fully generated encoding commands
		// Sort so that equality can be checked irrespective of element order.
		want := slices.Sorted(slices.Values([]string{
			"ffmpeg -i videos/clip01.mp4 -param1 x -y out/clip01_x264_param1_x.mp4",
			"ffmpeg -i videos/clip01.mp4 -param1 y -y out/clip01_x264_param1_y.mp4",
			"ffmpeg -i videos/clip02.mp4 -param1 x -y out/clip02_x264_param1_x.mp4",
			"ffmpeg -i videos/clip02.mp4 -param1 y -y out/clip02_x264_param1_y.mp4",
		}))
		got := slices.Sorted(slices.Values(gotCommands))
		it.Should(t, slices.Equal(got, want), "got = %v, want = %v", got, want)
	})
	t.Run("Should have expected output files", func(t *testing.T) {
		// And then I get correct expected output files
		want := slices.Sorted(slices.Values([]string{
			"out/clip01_x264_param1_x.out",
			"out/clip01_x264_param1_y.out",
			"out/clip02_x264_param1_x.out",
			"out/clip02_x264_param1_y.out",
		}))
		got := slices.Sorted(slices.Values(gotOutputFiles))
		it.Should(t, slices.Equal(got, want), "got = %v, want = %v", got, want)
	})
}

func Test_HappyPathPlanExecution(t *testing.T) {
	var plan Plan
	var pc PlanConfig
	outDir := t.TempDir()

	pc = PlanConfig{
		Inputs: []string{
			"../../testdata/video/testsrc01.mp4",
			"../../testdata/video/testsrc02.mp4",
		},
		Schemes: []Scheme{
			{
				"libx264 scheme1",
				`ffmpeg -i %INPUT% -an -c:v copy -y %OUTPUT%.mp4`,
			},
			{
				"libx264 scheme2",
				"ffmpeg -i %INPUT% -an -c:v copy -y %OUTPUT%.mkv",
			},
		},
	}
	wantResultCount := len(pc.Schemes) * len(pc.Inputs)

	plan = NewPlan(pc, outDir)
	gotResult, err := plan.Run()

	t.Run("Encoding result should have start and end time stamps", func(t *testing.T) {
		it.Should(t, gotResult.EndTime.After(gotResult.StartTime), "%v > %v", gotResult.EndTime, gotResult.StartTime)
	})
	t.Run("Encoding result should be available for each encoding command", func(t *testing.T) {
		gotCount := len(gotResult.RunResults)
		it.Should(t, gotCount == wantResultCount, "got = %v, want = %v", gotCount, wantResultCount)
	})
	t.Run("Encoding result should have ExitCodes", func(t *testing.T) {
		var gotExitCodes []int
		// Slice with exit codes of value 0
		wantExitCodes := make([]int, wantResultCount)
		for _, r := range gotResult.RunResults {
			gotExitCodes = append(gotExitCodes, r.ExitCode())
		}

		got := slices.Sorted(slices.Values(gotExitCodes))
		want := slices.Sorted(slices.Values(wantExitCodes))
		it.Should(t, slices.Equal(got, want), "got = %v, want = %v", got, want)
	})
	t.Run("Encoding result should have command stdout", func(t *testing.T) {
		// Test for existence of some known strings/markers of ffmpeg output
		markers := []string{
			"ffmpeg version",
			"Metadata:",
			"Duration:",
			"Input #0",
			"Output #0",
			"Stream #0",
			"time=",
			"bitrate=",
			"speed=",
		}

		for _, r := range gotResult.RunResults {
			gotOutput := r.Output()
			for _, m := range markers {
				it.Should(t, strings.Contains(gotOutput, m), "output missing marker %q", m)
			}
		}
	})
	t.Run("Encoding result should have no unexpected errors", func(t *testing.T) {
		if err != nil {
			for _, r := range gotResult.RunResults {
				if len(r.Errors) != 0 {
					t.Log(r.Name)
					t.Logf("Error: %v\n", r.Errors)
					t.Logf("Output: %v\n", r.Output())
				}
			}
			t.Fatal(err)
		}
	})
	t.Run("Encoding result should have correct source files", func(t *testing.T) {
		want := []string{
			"../../testdata/video/testsrc01.mp4",
			"../../testdata/video/testsrc02.mp4",
			"../../testdata/video/testsrc01.mp4",
			"../../testdata/video/testsrc02.mp4",
		}
		var got []string
		for _, r := range gotResult.RunResults {
			got = append(got, r.SourceFile)
		}
		it.Should(t, slices.Equal(got, want), "got = %v, want = %v", got, want)
	})
	t.Run("Encoding result should have correct compressed files", func(t *testing.T) {
		want := []string{
			fmt.Sprintf("%s/testsrc01_libx264_scheme1.mp4", outDir),
			fmt.Sprintf("%s/testsrc02_libx264_scheme1.mp4", outDir),
			fmt.Sprintf("%s/testsrc01_libx264_scheme2.mkv", outDir),
			fmt.Sprintf("%s/testsrc02_libx264_scheme2.mkv", outDir),
		}
		var got []string
		for _, r := range gotResult.RunResults {
			got = append(got, r.CompressedFile)
		}
		it.Should(t, slices.Equal(got, want), "got = %v, want = %v", got, want)
	})
	t.Run("Encoding result should have correct output files", func(t *testing.T) {
		want := []string{
			fmt.Sprintf("%s/testsrc01_libx264_scheme1.out", outDir),
			fmt.Sprintf("%s/testsrc02_libx264_scheme1.out", outDir),
			fmt.Sprintf("%s/testsrc01_libx264_scheme2.out", outDir),
			fmt.Sprintf("%s/testsrc02_libx264_scheme2.out", outDir),
		}
		var got []string
		for _, r := range gotResult.RunResults {
			got = append(got, r.OutputFile)
		}
		it.Should(t, slices.Equal(got, want), "got = %v, want = %v", got, want)
	})
	t.Run("Encoding result should have correct log files", func(t *testing.T) {
		want := []string{
			fmt.Sprintf("%s/testsrc01_libx264_scheme1.log", outDir),
			fmt.Sprintf("%s/testsrc02_libx264_scheme1.log", outDir),
			fmt.Sprintf("%s/testsrc01_libx264_scheme2.log", outDir),
			fmt.Sprintf("%s/testsrc02_libx264_scheme2.log", outDir),
		}
		var got []string
		for _, r := range gotResult.RunResults {
			got = append(got, r.LogFile)
		}
		it.Should(t, slices.Equal(got, want), "got = %v, want = %v", got, want)
	})
	t.Run("Compressed output file(s) should exist", func(t *testing.T) {
		for _, c := range plan.Commands {
			it.Should(t, testutil.FileExists(c.CompressedFile), "file should exist: %v", c.CompressedFile)
		}
	})
	t.Run("Command output file(s) should exist", func(t *testing.T) {
		for _, c := range plan.Commands {
			it.Should(t, testutil.FileExists(c.OutputFile), "file should exist: %v", c.OutputFile)
		}
	})
	t.Run("Encoding result should have usage stats", func(t *testing.T) {
		for _, r := range gotResult.RunResults {
			t.Logf("Checking Stats for: %s\n", r.Name)
			gotStats := r.Stats
			// Although individually these can be 0, Stime + Utime should be safely
			// asserted to be > 0
			it.Should(t, gotStats.Stime+gotStats.Utime > 0, "got = %v, want > 0", gotStats.Stime+gotStats.Utime)
			// MaxRss should always be > 0
			it.Should(t, gotStats.MaxRss > 0, "got = %v, want > 0", gotStats.MaxRss)
			// Elapsed should always be > 0
			it.Should(t, gotStats.Elapsed > 0, "got = %v, want > 0", gotStats.Elapsed)
			// CPUPercent() should always be > 0
			it.Should(t, gotStats.CPUPercent() > 0, "got = %v, want > 0", gotStats.CPUPercent())
		}
	})
	t.Run("Encoding result should have Duration", func(t *testing.T) {
		var gotDurations []float64
		// This depends on video Inputs and Scheme combination
		wantDurations := []float64{1, 10, 1, 10}
		for _, r := range gotResult.RunResults {
			gotDurations = append(gotDurations, r.VideoDuration)
		}
		it.Should(t, slices.Equal(gotDurations, wantDurations), "got = %v, want = %v", gotDurations, wantDurations)
	})
	t.Run("Encoding result should have average encoding speed", func(t *testing.T) {
		for _, r := range gotResult.RunResults {
			it.Should(t, r.AvgEncodingSpeed > 0, "got = %v, want > 0", r.AvgEncodingSpeed)
		}
	})
}

func TestNegativeEncodingPlanRunWitOutputOverflow(t *testing.T) {
	outDir := t.TempDir()
	planConfig := PlanConfig{
		Inputs: []string{"not_important"},
		Schemes: []Scheme{
			// Unix yes should be fast enough to generate output that overflows
			{"large output", "../../testdata/helpers/stderr yes"},
		},
	}
	// 128 + 13 (SIGPIPE)
	wantExitCode := 141

	// Given a Plan
	plan := NewPlan(planConfig, outDir)

	// When I do an unsuccessful Run of a Plan
	gotResult, err := plan.Run()
	it.Must(t, err != nil, "Should have error for unsuccessful Run")

	gotExitCode := gotResult.RunResults[0].ExitCode()
	it.Should(t, gotExitCode == wantExitCode, "Should have correct ExitCode when Run fails: got = %v, want = %v", gotExitCode, wantExitCode)
}

func TestNegativeEncodingPlanResults(t *testing.T) {
	outDir := t.TempDir()
	planConfig := PlanConfig{
		Inputs: []string{"../../testdata/video/testsrc01.mp4"},
		Schemes: []Scheme{
			{"failing", "ls some_gibberish %INPUT% %OUTPUT%"},
			// For the sake of completeness - have a successful run also
			{"passing", "../../testdata/helpers/stderr cp -v %INPUT% %OUTPUT%.mp4"},
		},
	}
	// Given a Plan
	plan := NewPlan(planConfig, outDir)
	// When I do an unsuccessful Run of a Plan
	gotResult, err := plan.Run()

	t.Run("Should have error for unsuccessful Run", func(t *testing.T) {
		it.Must(t, err != nil, "Should have error for unsuccessful Run")
	})
	t.Run("Should have correct ExitCode (!=0) when Run fails", func(t *testing.T) {
		gotExitCode := gotResult.RunResults[0].ExitCode()
		it.Should(t, gotExitCode != 0, "got = %v, want non-zero", gotExitCode)
	})
	t.Run("Should have expected Error when Run fails", func(t *testing.T) {
		wantError := "exit status"
		// A bit dirty way to "glue" together multiple errors.
		gotError := fmt.Sprintf("%v", gotResult.RunResults[0].Errors)
		it.Should(t, strings.Contains(gotError, wantError), "got = %v, want to contain %v", gotError, wantError)
	})
	t.Run("Successful runs should not be influenced by unsuccessful", func(t *testing.T) {
		wantExitCode := 0
		gotExitCode := gotResult.RunResults[1].ExitCode()
		it.Should(t, gotExitCode == wantExitCode, "got = %v, want = %v", gotExitCode, wantExitCode)

		errors := gotResult.RunResults[1].Errors
		it.Should(t, errors == nil, "Successful run had unexpected Errors: %v", errors)
	})
}

func TestSchemeUnmarshalJSON(t *testing.T) {
	tests := map[string]struct {
		want  Scheme
		given []byte
	}{
		"Empty JSON": {
			given: []byte(`{}`),
			want:  Scheme{},
		},
		"Wrong JSON": {
			given: []byte(`{"aaa": ""}`),
			want:  Scheme{},
		},
		"CommandTpl is null": {
			given: []byte(`{"Name": "name", "CommandTpl": null}`),
			want:  Scheme{Name: "name", CommandTpl: ""},
		},
		"CommandTpl is empty array": {
			given: []byte(`{"Name": "name", "CommandTpl": []}`),
			want:  Scheme{Name: "name", CommandTpl: ""},
		},
		"CommandTpl with single element": {
			given: []byte(`{"Name": "name", "CommandTpl": ["a"]}`),
			want:  Scheme{Name: "name", CommandTpl: "a"},
		},
		"CommandTpl with multiple elements": {
			given: []byte(`{"Name": "name", "CommandTpl": ["aa", "bbb", " ccc ", "ddd"]}`),
			want:  Scheme{Name: "name", CommandTpl: "aabbb ccc ddd"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var got Scheme
			err := json.Unmarshal(tc.given, &got)
			it.Should(t, err == nil, "Unexpected error: got = %v, want = nil", err)
			it.Should(t, got == tc.want, "Scheme.UnmarshalJSON mismatch: got = %v, want = %v", got, tc.want)
		})
	}
}

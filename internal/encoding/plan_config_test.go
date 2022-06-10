// Copyright ©2022 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Encoding plan configuration related tests.

package encoding

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNewPlanConfigFromJSON(t *testing.T) {
	tests := map[string]struct {
		err   error
		want  PlanConfig
		given []byte
	}{
		"Positive": {
			given: []byte(`{
				"OutDir": "out",
				"Inputs": [
					"src/vid1.mp4",
					"src/vid2.mp4"
				],
				"Schemes": [
					{
						"Name": "sc1",
						"CommandTpl": ["sc1 ",  "command template"]
					},
					{
						"Name": "sc2",
						"CommandTpl": ["sc2 command ", "template"]
					}
				]
			}`),
			want: PlanConfig{
				OutDir: "out",
				Inputs: []string{
					"src/vid1.mp4",
					"src/vid2.mp4",
				},
				Schemes: []Scheme{
					{"sc1", "sc1 command template"},
					{"sc2", "sc2 command template"},
				},
			},
			err: nil,
		},
		// Should this be positive?!
		"Positive incomplete JSON": {
			given: []byte(`{ "OutDir": "out" }`),
			want:  PlanConfig{OutDir: "out"},
			err:   nil,
		},
		"Negative invalid JSON": {
			given: []byte("]"),
			want:  PlanConfig{},
			err:   &json.SyntaxError{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := NewPlanConfigFromJSON(tc.given)

			if tc.err == nil {
				// Positive scenario case when error should be absent (nil).
				if !errors.Is(err, tc.err) {
					t.Fatalf("Unexpected error. want %T, got %T", tc.err, err)
				}
			} else {
				if gotE, wantE := reflect.TypeOf(err), reflect.TypeOf(tc.err); gotE != wantE {
					t.Errorf("Error type mismatch want: %v, got: %v\n", wantE, gotE)
				}
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("PlanConfig mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestPlanConfigIsValid(t *testing.T) {
	pc := PlanConfig{
		OutDir:  ".",
		Inputs:  []string{"../../testdata/video/testsrc01.mp4"},
		Schemes: []Scheme{{}},
	}
	got, err := pc.IsValid()

	t.Run("Should successfully validate PlanConfig", func(t *testing.T) {
		if !got {
			t.Errorf("PlanConfig.IsValid() validation failed for PlanConfig: %v", pc)
		}
	})
	t.Run("Should have no errors", func(t *testing.T) {
		if err != nil {
			if e, ok := err.(*PlanConfigError); ok {
				t.Errorf("Got PlanConfigError: %v", e)
			}
			t.Errorf("PlanConfig.IsValid() unexpected error: %v", err)
		}
	})
}

func TestNegativePlanConfigIsValid(t *testing.T) {
	wantErrorMsg := "validation error"
	tests := map[string]struct {
		given       PlanConfig
		wantReasons []string
	}{
		"Negative nil value": {
			given: PlanConfig{},
			wantReasons: []string{
				"Inputs missing", "Schemes missing", "OutDir missing",
			},
		},
		"Negative Schemes missing": {
			given: PlanConfig{
				OutDir: ".",
				Inputs: []string{"../../testdata/video/testsrc01.mp4"},
			},
			wantReasons: []string{
				"Schemes missing",
			},
		},
		"Negative Inputs missing": {
			given: PlanConfig{
				OutDir:  ".",
				Schemes: []Scheme{{}},
			},
			wantReasons: []string{
				"Inputs missing",
			},
		},
		"Negative duplicate Inputs": {
			given: PlanConfig{
				OutDir:  ".",
				Schemes: []Scheme{{}},
				Inputs:  []string{"../../testdata/video/testsrc01.mp4", "../../testdata/video/testsrc01.mp4"},
			},
			wantReasons: []string{
				"Duplicate inputs detected",
			},
		},
		"Negative empty OutDir": {
			given: PlanConfig{
				OutDir:  "",
				Inputs:  []string{"../../testdata/video/testsrc01.mp4"},
				Schemes: []Scheme{{}},
			},
			wantReasons: []string{
				"OutDir missing",
			},
		},
		"Negative wrong file in Inputs": {
			given: PlanConfig{
				OutDir:  ".",
				Inputs:  []string{"no_existent_file"},
				Schemes: []Scheme{{}},
			},
			wantReasons: []string{
				"stat no_existent_file: no such file or directory",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := tc.given.IsValid()
			if !strings.HasPrefix(err.Error(), wantErrorMsg) {
				t.Errorf("PlanConfig.IsValid() error: want=%s*, got=%v", wantErrorMsg, err)
			}
			if got {
				t.Errorf("PlanConfig.IsValid() returned %v, want false", got)
			}
			// Cast error in order to check Reasons().
			gotErr, ok := err.(*PlanConfigError)
			if !ok {
				t.Errorf("PlanConfig.IsValid() returned unexpected error type, want PlanConfigError, got %T", err)
			}
			if diff := cmp.Diff(tc.wantReasons, gotErr.Reasons()); diff != "" {
				t.Errorf("PlanConfigError reasons mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestHasDuplicatesTable(t *testing.T) {
	tests := map[string]struct {
		given []string
		want  bool
	}{
		"No duplicates": {
			given: []string{"aaa", "bbb", "ccc", "ddd"},
			want:  false,
		},
		"No duplicates empty": {
			given: []string{},
			want:  false,
		},
		"With duplicates": {
			given: []string{"aaa", "bbb", "ccc", "aaa", "ddd"},
			want:  true,
		},
		"With duplicate empty strings": {
			given: []string{"", "bbb", "ccc", "", "ddd"},
			want:  true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := hasDuplicates(tc.given)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("HasDuplicates mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

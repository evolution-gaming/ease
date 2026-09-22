// Copyright ©2022 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Encoding plan configuration related tests.

package encoding

import (
	"encoding/json"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/evolution-gaming/ease/internal/it"
)

func TestNewPlanConfigFromJSON(t *testing.T) {
	tests := map[string]struct {
		want  PlanConfig
		given []byte
		// For positive scenario leave check function nil.
		checkErr func(err error) bool
	}{
		"Positive": {
			given: []byte(`{
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
				Inputs: []string{
					"src/vid1.mp4",
					"src/vid2.mp4",
				},
				Schemes: []Scheme{
					{"sc1", "sc1 command template"},
					{"sc2", "sc2 command template"},
				},
			},
		},
		// Should this be positive?!
		"Positive incomplete JSON": {
			given: []byte(`{ "Inputs": ["input1"]}`),
			want:  PlanConfig{Inputs: []string{"input1"}},
		},
		"Negative invalid JSON": {
			given: []byte("]"),
			want:  PlanConfig{},
			checkErr: func(err error) bool {
				var errT *json.SyntaxError
				return errors.As(err, &errT)
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := NewPlanConfigFromJSON(tc.given)

			if tc.checkErr == nil {
				// Positive scenario case when error should be absent (nil).
				it.Should(t, err == nil, "Unexpected error: got = %v, want = nil", err)
			} else {
				// Negative scenario with expected non-nil error.
				it.Should(t, tc.checkErr(err), "Unexpected error type: %T", err)
			}
			it.Should(t, reflect.DeepEqual(got, tc.want), "got = %v, want = %v", got, tc.want)
		})
	}
}

func TestPlanConfigIsValid(t *testing.T) {
	pc := PlanConfig{
		Inputs:  []string{"../../testdata/video/testsrc01.mp4"},
		Schemes: []Scheme{{}},
	}
	validState, err := pc.IsValid()
	it.Should(t, validState, "PlanConfig validation failed")
	it.Should(t, err == nil, "Unexpected error: got = %v, want = nil", err)
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
				"Inputs missing", "Schemes missing",
			},
		},
		"Negative Schemes missing": {
			given: PlanConfig{
				Inputs: []string{"../../testdata/video/testsrc01.mp4"},
			},
			wantReasons: []string{
				"Schemes missing",
			},
		},
		"Negative Inputs missing": {
			given: PlanConfig{
				Schemes: []Scheme{{}},
			},
			wantReasons: []string{
				"Inputs missing",
			},
		},
		"Negative duplicate Inputs": {
			given: PlanConfig{
				Schemes: []Scheme{{}},
				Inputs:  []string{"../../testdata/video/testsrc01.mp4", "../../testdata/video/testsrc01.mp4"},
			},
			wantReasons: []string{
				"Duplicate inputs detected",
			},
		},
		"Negative wrong file in Inputs": {
			given: PlanConfig{
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
			validState, err := tc.given.IsValid()
			it.Must(t, err != nil, "Got nil instead of error")
			it.Should(t, strings.Contains(err.Error(), wantErrorMsg), "expected error message missing: %v", wantErrorMsg)
			it.Should(t, validState == false, "got = %v, want = %v", validState, false)

			// Cast error in order to check Reasons().
			gotErr, ok := err.(*PlanConfigError)
			it.Must(t, ok, "Unexpected error type, want PlanConfigError, got %T", err)
			it.Should(t, slices.Equal(tc.wantReasons, gotErr.Reasons()),
				"Reasons mismatch: got = %v, want = %v", gotErr.Reasons(), tc.wantReasons)
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
			it.Should(t, got == tc.want, "got = %v, want = %v", got, tc.want)
		})
	}
}

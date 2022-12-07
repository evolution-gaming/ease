// Copyright ©2022 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Encoding plan configuration related tests.

package encoding

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPlanConfigFromJSON(t *testing.T) {
	tests := map[string]struct {
		err   error
		want  PlanConfig
		given []byte
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
			err: nil,
		},
		// Should this be positive?!
		"Positive incomplete JSON": {
			given: []byte(`{ "Inputs": ["input1"]}`),
			want:  PlanConfig{Inputs: []string{"input1"}},
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
				assert.NoError(t, err)
			} else {
				// Negative scenario with expected non-nil error.
				gotE := reflect.TypeOf(err)
				wantE := reflect.TypeOf(tc.err)
				assert.Equal(t, wantE, gotE)
			}
			assert.Equal(t, tc.want, got, "PlanConfig mismatch")
		})
	}
}

func TestPlanConfigIsValid(t *testing.T) {
	pc := PlanConfig{
		Inputs:  []string{"../../testdata/video/testsrc01.mp4"},
		Schemes: []Scheme{{}},
	}
	validState, err := pc.IsValid()
	assert.True(t, validState)
	assert.NoError(t, err)
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
			assert.ErrorContains(t, err, wantErrorMsg)
			assert.False(t, validState)

			// Cast error in order to check Reasons().
			gotErr, ok := err.(*PlanConfigError)
			assert.Truef(t, ok, "Unexpected error type, want PlanConfigError, got %T", err)
			assert.Equal(t, tc.wantReasons, gotErr.Reasons())
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
			assert.Equal(t, tc.want, got)
		})
	}
}

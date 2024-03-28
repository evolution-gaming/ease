// Copyright ©2022 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Tests for reusable parts of ease application and subcommand infrastructure.
package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_all_Positive(t *testing.T) {
	floatTests := map[string]struct {
		cmp   func(x float64) bool
		given []float64
		want  bool
	}{
		"all match": {
			given: []float64{1.0, 1.0, 1.0, 1.0},
			cmp:   func(x float64) bool { return x == 1 },
			want:  true,
		},
		"some don't match": {
			given: []float64{1, 0.9999999999, 1, 1},
			cmp:   func(x float64) bool { return x == 1 },
			want:  false,
		},
		"empty slice": {
			given: []float64{},
			cmp:   func(x float64) bool { return x == 0 },
			want:  false,
		},
		"nil slice": {
			given: nil,
			cmp:   func(x float64) bool { return x == 0 },
			want:  false,
		},
	}

	stringTests := map[string]struct {
		cmp   func(x string) bool
		given []string
		want  bool
	}{
		"all match": {
			given: []string{"foo", "foo", "foo"},
			cmp:   func(x string) bool { return x == "foo" },
			want:  true,
		},
		"some don't match": {
			given: []string{"foo", "foo", "bar", "foo", "baz"},
			cmp:   func(x string) bool { return x == "foo" },
			want:  false,
		},
		"empty strings": {
			given: []string{"", "", ""},
			cmp:   func(x string) bool { return x == "" },
			want:  true,
		},
		"empty slice": {
			given: []string{},
			cmp:   func(x string) bool { return x == "" },
			want:  false,
		},
		"nil slice": {
			given: nil,
			cmp:   func(x string) bool { return x == "" },
			want:  false,
		},
	}

	t.Run("float type tests", func(t *testing.T) {
		for name, tc := range floatTests {
			t.Run(name, func(t *testing.T) {
				assert.Equal(t, tc.want, all(tc.given, tc.cmp))
			})
		}
	})

	t.Run("string type tests", func(t *testing.T) {
		for name, tc := range stringTests {
			t.Run(name, func(t *testing.T) {
				assert.Equal(t, tc.want, all(tc.given, tc.cmp))
			})
		}
	})
}

func Test_parseFraction(t *testing.T) {
	tests := map[string]struct {
		given   string
		want    float64
		wantErr bool
	}{
		"valid1": {
			given:   "2997/125",
			want:    23.976,
			wantErr: false,
		},
		"valid2": {
			given:   "25/1",
			want:    25,
			wantErr: false,
		},
		"valid3": {
			given:   "24",
			want:    24,
			wantErr: false,
		},
		"invalid1": {
			given:   "/23",
			want:    0,
			wantErr: true,
		},
		"invalid2": {
			given:   "23/",
			want:    0,
			wantErr: true,
		},
		"invalid3": {
			given:   "/",
			want:    0,
			wantErr: true,
		},
		"invalid4": {
			given:   "1/0",
			want:    0,
			wantErr: true,
		},
		"invalid5": {
			given:   "12.5/1",
			want:    0,
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := parseFraction(tc.given)
			switch tc.wantErr {
			case true:
				assert.Error(t, err)
			default:
				assert.NoError(t, err)
			}
			assert.InDelta(t, tc.want, got, 1e-9)
		})
	}
}

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
		given []float64
		cmp   float64
		want  bool
	}{
		"all match": {
			given: []float64{1.0, 1.0, 1.0, 1.0},
			cmp:   1,
			want:  true,
		},
		"some don't match": {
			given: []float64{1, 0.9999999999, 1, 1},
			cmp:   1,
			want:  false,
		},
		"empty slice": {
			given: []float64{},
			cmp:   0,
			want:  false,
		},
		"nil slice": {
			given: nil,
			cmp:   0,
			want:  false,
		},
	}

	stringTests := map[string]struct {
		cmp   string
		given []string
		want  bool
	}{
		"all match": {
			given: []string{"foo", "foo", "foo"},
			cmp:   "foo",
			want:  true,
		},
		"some don't match": {
			given: []string{"foo", "foo", "bar", "foo", "baz"},
			cmp:   "foo",
			want:  false,
		},
		"empty strings": {
			given: []string{"", "", ""},
			cmp:   "",
			want:  true,
		},
		"empty slice": {
			given: []string{},
			cmp:   "",
			want:  false,
		},
		"nil slice": {
			given: nil,
			cmp:   "",
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

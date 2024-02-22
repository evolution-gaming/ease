// Copyright ©2022 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Tests for reusable parts of ease application and subcommand infrastructure.
package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_extractSourceData(t *testing.T) {
	given := parseReportFile("testdata/encoding_artifacts/report.json")
	want := map[string]sourceData{
		"out/testsrc01_libx264.mp4": {
			CompressedFile: "out/testsrc01_libx264.mp4",
			WorkDir:        "/tmp",
			VqmResultFile:  "out/testsrc01_libx264_vqm.json",
		},
		"out/testsrc01_libx265.mp4": {
			CompressedFile: "out/testsrc01_libx265.mp4",
			WorkDir:        "/tmp",
			VqmResultFile:  "out/testsrc01_libx265_vqm.json",
		},
		"out/testsrc02_libx264.mp4": {
			CompressedFile: "out/testsrc02_libx264.mp4",
			WorkDir:        "/tmp",
			VqmResultFile:  "out/testsrc02_libx264_vqm.json",
		},
		"out/testsrc02_libx265.mp4": {
			CompressedFile: "out/testsrc02_libx265.mp4",
			WorkDir:        "/tmp",
			VqmResultFile:  "out/testsrc02_libx265_vqm.json",
		},
	}

	got := extractSourceData(given)
	assert.Equal(t, want, got)
}

func Test_parseReportFile(t *testing.T) {
	got := parseReportFile("testdata/encoding_artifacts/report.json")
	t.Log("Should have RunResults")
	assert.Len(t, got.EncodingResult.RunResults, 4)

	t.Log("Should have VQMResults")
	assert.Len(t, got.VQMResults, 4)
}

func Test_report_WriteJSON(t *testing.T) {
	// Do the round-trip of JOSN report unmarshalling-marshalling.
	reportFile := "testdata/encoding_artifacts/report.json"
	parsedReport := parseReportFile(reportFile)

	var got bytes.Buffer
	err := parsedReport.WriteJSON(&got)
	assert.NoError(t, err)

	want, err := os.ReadFile(reportFile)
	assert.NoError(t, err)
	wantStr := strings.TrimRight(string(want), "\n")

	assert.Equal(t, wantStr, got.String())
}

func Test_csvReport_writeCSV(t *testing.T) {
	// Create report from fixture data.
	reportFile := "testdata/encoding_artifacts/report.json"
	parsedReport := parseReportFile(reportFile)

	csvReport, err := newCsvReport(parsedReport)
	assert.NoError(t, err)

	var b bytes.Buffer
	err = csvReport.WriteCSV(&b)
	assert.NoError(t, err)

	assert.Len(t, b.Bytes(), 1332)
}

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

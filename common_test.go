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
	parsedReport.WriteJSON(&got)

	want, err := os.ReadFile(reportFile)
	assert.NoError(t, err)
	wantStr := strings.TrimRight(string(want), "\n")

	assert.Equal(t, wantStr, got.String())
}

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

	"github.com/google/go-cmp/cmp"
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
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("extractSourceData result mismatch (-want +got):\n%s", diff)
	}
}

func Test_parseReportFile(t *testing.T) {
	got := parseReportFile("testdata/encoding_artifacts/report.json")
	t.Run("Should have RunResults", func(t *testing.T) {
		if len(got.EncodingResult.RunResults) != 4 {
			t.Error("Expecting 4 elements in RunResults")
		}
	})
	t.Run("Should have VQMResults", func(t *testing.T) {
		if len(got.VQMResults) != 4 {
			t.Error("Expecting 4 elements in VQMResults")
		}
	})
}

func Test_report_WriteJSON(t *testing.T) {
	// Do the round-trip of JOSN report unmarshalling-marshalling.
	reportFile := "testdata/encoding_artifacts/report.json"
	parsedReport := parseReportFile(reportFile)

	var got bytes.Buffer
	parsedReport.WriteJSON(&got)

	want, err := os.ReadFile(reportFile)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	wantStr := strings.TrimRight(string(want), "\n")

	if diff := cmp.Diff(wantStr, got.String()); diff != "" {
		t.Errorf("JSON roundtrip failed (-want +got):\n%s", diff)
	}
}

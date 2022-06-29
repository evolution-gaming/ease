// Copyright ©2022 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package logging

import (
	"log"
	"regexp"
	"strings"
	"testing"
)

func TestUnformattedLogging(t *testing.T) {
	tests := map[string]struct {
		given   string
		want    *regexp.Regexp
		logFunc func(...interface{})
		logger  *log.Logger
	}{
		"Simple Info": {
			given:   "info message",
			want:    regexp.MustCompile("INFO: .*info message"),
			logFunc: Info,
			logger:  InfoLogger,
		},
		"Simple Debug": {
			given:   "debug message",
			want:    regexp.MustCompile("DEBUG: .*debug message"),
			logFunc: Debug,
			logger:  DebugLogger,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var out strings.Builder
			tc.logger.SetOutput(&out)
			tc.logFunc(tc.given)
			got := out.String()
			if !tc.want.MatchString(got) {
				t.Errorf("Log message not found (-want/+got)\n\t-%s\n\t+%s", tc.want.String(), got)
			}
		})
	}
}

func TestFormattedLogging(t *testing.T) {
	tests := map[string]struct {
		given1  string
		given2  string
		want    *regexp.Regexp
		format  string
		logFunc func(string, ...interface{})
		logger  *log.Logger
	}{
		"Complex Info": {
			given1:  "info message 1",
			given2:  "info message 2",
			want:    regexp.MustCompile("INFO: .*info message 1 -- info message 2"),
			format:  "%s -- %s",
			logFunc: Infof,
			logger:  InfoLogger,
		},
		"Complex Debug": {
			given1:  "debug message 1",
			given2:  "debug message 2",
			format:  "%s -- %s",
			want:    regexp.MustCompile("DEBUG: .*debug message 1 -- debug message 2"),
			logFunc: Debugf,
			logger:  DebugLogger,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var out strings.Builder
			tc.logger.SetOutput(&out)
			tc.logFunc(tc.format, tc.given1, tc.given2)
			got := out.String()
			if !tc.want.MatchString(got) {
				t.Errorf("Log message not found (-want/+got)\n\t-%s\n\t+%s", tc.want.String(), got)
			}
		})
	}
}

func Test_EnableInfoLogger(t *testing.T) {
	t.Run("Enabling info logger should set log writer", func(t *testing.T) {
		before := InfoLogger.Writer()
		EnableInfoLogger()
		after := InfoLogger.Writer()

		if after != defaultOutput {
			t.Errorf("InfoLogger writer mismatch (-want +got):\n\t-%#v\n\t+%#v",
				defaultOutput, after)
		}

		if after == before {
			t.Error("EnableInfoLogger() had no effect: before and after writers are the same")
		}
	})
}

func Test_EnableDebugLogger(t *testing.T) {
	t.Run("Enabling debug logger should set log writer", func(t *testing.T) {
		before := DebugLogger.Writer()
		EnableDebugLogger()
		after := DebugLogger.Writer()

		if after != defaultOutput {
			t.Errorf("DebugLogger writer mismatch (-want +got):\n\t-%#v\n\t+%#v",
				defaultOutput, after)
		}

		if after == before {
			t.Error("EnableDebugLogger() had no effect: before and after writers are the same")
		}
	})
}

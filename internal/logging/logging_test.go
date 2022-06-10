// Copyright ©2022 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package logging_test

import (
	"log"
	"regexp"
	"strings"
	"testing"

	"github.com/evolution-gaming/ease/internal/logging"
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
			logFunc: logging.Info,
			logger:  logging.InfoLogger,
		},
		"Simple Debug": {
			given:   "debug message",
			want:    regexp.MustCompile("DEBUG: .*debug message"),
			logFunc: logging.Debug,
			logger:  logging.DebugLogger,
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
			logFunc: logging.Infof,
			logger:  logging.InfoLogger,
		},
		"Complex Debug": {
			given1:  "debug message 1",
			given2:  "debug message 2",
			format:  "%s -- %s",
			want:    regexp.MustCompile("DEBUG: .*debug message 1 -- debug message 2"),
			logFunc: logging.Debugf,
			logger:  logging.DebugLogger,
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

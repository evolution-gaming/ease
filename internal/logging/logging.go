// Copyright ©2022 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Poor man's logging. Implements 2-level loggers for Info and Debug. Minimal
// wrap around standard library's "log" package.
package logging

import (
	"fmt"
	"io"
	"log"
)

var (
	defaultOutput io.Writer = log.Default().Writer()
	debugFlags              = log.Ldate | log.Ltime | log.Lshortfile
	infoFlags               = log.Ldate | log.Ltime
	// Each log-level logger should be explicitly enabled via call to Enable*Logger().
	DebugLogger = log.New(io.Discard, debugPrefix, debugFlags)
	InfoLogger  = log.New(io.Discard, infoPrefix, infoFlags)
)

const (
	debugPrefix = "DEBUG: "
	infoPrefix  = "INFO: "
	calldepth   = 2
)

// EnableInfoLogger helper function to explicitly enable InfoLogger.
func EnableInfoLogger() {
	InfoLogger.SetOutput(defaultOutput)
}

// EnableDebugLogger helper function to explicitly enable DebugLogger.
func EnableDebugLogger() {
	DebugLogger.SetOutput(defaultOutput)
}

func Info(v ...interface{}) {
	InfoLogger.Output(calldepth, fmt.Sprint(v...))
}

func Infof(format string, v ...interface{}) {
	InfoLogger.Output(calldepth, fmt.Sprintf(format, v...))
}

func Debug(v ...interface{}) {
	DebugLogger.Output(calldepth, fmt.Sprint(v...))
}

func Debugf(format string, v ...interface{}) {
	DebugLogger.Output(calldepth, fmt.Sprintf(format, v...))
}

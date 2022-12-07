// Copyright ©2022 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Application version string related functionality.
//
// Implementation should work for case when application binary is built via "go build" and
// version injection via "ldflags" as well as when binary is installed via "go install" in
// which case debug.BuildInfo is used to pull relevant version information.

package main

import (
	"fmt"
	"os"
	"runtime/debug"
	"time"
)

// Value injected during build with -ldflags="-X main.version={ver}".
var (
	version string
	vInfo   versionInfo
)

func init() {
	// A case when version is passed in via -ldflags="-X main.version=xxx"
	if version != "" {
		vInfo.version = version
	}

	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}

	if vInfo.version == "" {
		vInfo.version = bi.Main.Version
	}

	for _, v := range bi.Settings {
		switch v.Key {
		case "vcs.revision":
			vInfo.revision = v.Value
		case "vcs.time":
			vInfo.time, _ = time.Parse(time.RFC3339, v.Value)
		}
	}
}

// versionInfo is struct that includes relevant version information.
type versionInfo struct {
	time     time.Time
	version  string
	revision string
}

func (v versionInfo) String() string {
	if v.revision == "" {
		return v.version
	}
	return fmt.Sprintf("%s %s", v.version, v.revision)
}

func printVersion() {
	fmt.Fprintln(os.Stderr, vInfo)
}

// Copyright ©2022 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// ease tool's new-plan subcommand implementation.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/evolution-gaming/ease/internal/encoding"
)

// inputFiles implements flag.Value interface.
type inputFiles []string

func (i *inputFiles) String() string {
	return strings.Join(*i, ", ")
}

func (i *inputFiles) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func CreateNewPlanCommand() *NewPlanApp {
	longHelp := `Subcommand "new-plan" helps create a new plan configuration file template.

Examples:

  ease new-plan -i path/to/input/video.mp4 -o plan.json
  ease new-plan -i video1.mp4 -i video2.mp4 -o plan.json`

	app := &NewPlanApp{
		fs: flag.NewFlagSet("new-plan", flag.ContinueOnError),
	}
	app.fs.StringVar(&app.flOutFile, "o", "", "Output file (stdout by default).")
	app.fs.Var(&app.flInputFiles, "i", "Source video files. Use multiple times for multiple files.")

	app.fs.Usage = func() {
		printSubCommandUsage(longHelp, app.fs)
	}

	return app
}

type NewPlanApp struct {
	// FlagSet instance
	fs *flag.FlagSet
	// Output file to save plot to
	flOutFile string
	// Video input files
	flInputFiles inputFiles
}

func (a *NewPlanApp) Run(args []string) error {
	if err := a.fs.Parse(args); err != nil {
		return &AppError{
			msg:      "usage error",
			exitCode: 2,
		}
	}

	// In case no input video provided we will use some placeholder string.
	if len(a.flInputFiles) == 0 {
		a.flInputFiles = []string{"path/to/source/video.mp4"}
	}

	// Create a PlanConfig instance and populate it with some data. From this we shall
	// crate a JSON plan.
	pc := encoding.PlanConfig{}
	pc.Inputs = a.flInputFiles
	pc.Schemes = []encoding.Scheme{
		{
			Name:       "encode1",
			CommandTpl: "ffmpeg -i %INPUT% -c:v libx264 -preset fast -crf 23 %OUTPUT%.mp4",
		},
		{
			Name:       "encode2",
			CommandTpl: "ffmpeg -i %INPUT% -c:v libx264 -preset faster -crf 25 %OUTPUT%.mp4",
		},
	}

	var out io.Writer
	switch a.flOutFile {
	case "":
		out = os.Stdout
	default:
		fd, err := os.Create(a.flOutFile)
		if err != nil {
			return &AppError{
				msg:      fmt.Sprintf("output file error: %s", err),
				exitCode: 1,
			}
		}
		out = fd
	}

	e := json.NewEncoder(out)
	e.SetIndent("", "  ")
	if err := e.Encode(pc); err != nil {
		return &AppError{
			msg:      "JSON marshal error",
			exitCode: 1,
		}
	}

	return nil
}

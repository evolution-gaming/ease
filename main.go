// Copyright ©2022 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Main entrypoint for ease application

package main

import (
	"fmt"
	"os"

	"github.com/evolution-gaming/ease/internal/logging"
)

// root represents top level of ease command, including dispatching to subcommands.
func root(args []string) error {
	usage := `Ease - Encoder Evaluation Suite

Usage:

    ease <command> [arguments] [-h|-help]

The commands are:

    run         batch execute encodings according to "encoding plan"
    vqmplot     create plot for given metric from libvmaf JSON report
    bitrate     create bitrate plot of given video file
    dump-conf   output actual application configuration
    version     print ease version and exit

Use "ease <command> -h|-help" for more information about command.`

	if len(args) < 1 {
		fmt.Println(usage)
		return &AppError{msg: "please, specify command", exitCode: 2}
	}

	switch args[0] {
	case "run":
		return CreateRunCommand().Run(args[1:])
	case "vqmplot":
		return CreateVQMPlotCommand().Run(args[1:])
	case "bitrate":
		return CreateBitrateCommand().Run(args[1:])
	case "dump-conf", "dump":
		return CreateDumpConfCommand().Run(args[1:])
	case "version":
		printVersion()
		return nil
	case "-h", "-help", "--help", "?":
		fmt.Println(usage)
		return &AppError{
			exitCode: 2,
		}
	default:
		// No commands were matched at this point, so bail out with default usage message.
		fmt.Println(usage)
		return &AppError{
			msg:      "unknown command/flag",
			exitCode: 2,
		}
	}
}

func main() {
	// Enable info logger by default and early enough.
	logging.EnableInfoLogger()

	if err := root(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		switch e := err.(type) {
		case *AppError:
			os.Exit(e.ExitCode())
		default:
			os.Exit(1)
		}
	}
	os.Exit(0)
}

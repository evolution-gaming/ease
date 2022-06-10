// Copyright ©2022 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Main entrypoint for ease application

package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/evolution-gaming/ease/internal/logging"
)

var commandName = "ease"

func printVersion() {
	fmt.Fprintln(os.Stderr, vInfo)
}

// root represents top level of ease command, including dispatching to subcommands.
func root() error {
	// top level / global flags
	var flVersion bool
	var flDebug bool
	fs := flag.NewFlagSet(commandName, flag.ExitOnError)
	fs.BoolVar(&flVersion, "version", false, "Print version")
	fs.BoolVar(&flDebug, "debug", false, "Run in debug mode")

	// Register all subcommands here.
	subCmds := []Commander{
		CreateEncodeCommand(),
		CreateAnalyseCommand(),
		CreateBitrateCommand(),
		CreateVQMPlotCommand(),
	}

	// Custom Usage function that also calls into subcommand help output.
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "NAME\n  %s - Encoder Automation Suite\n\n", commandName)
		fmt.Fprintf(fs.Output(), "USAGE\n  %s [global flags] <sub-command> <arguments> [-h|-help]\n\n", commandName)
		fmt.Fprintf(fs.Output(), "GLOBAL FLAGS\n\n")

		fs.PrintDefaults()
		fmt.Fprintln(fs.Output())

		// Define subcommand help.
		var subCmdNames []string
		for _, c := range subCmds {
			subCmdNames = append(subCmdNames, c.Name())
		}
		fmt.Fprintf(fs.Output(), "SUB-COMMANDS\n\n")
		fmt.Fprintf(fs.Output(), "  %s\n", strings.Join(subCmdNames, ", "))
		fmt.Fprintln(fs.Output())
		for _, c := range subCmds {
			c.Help()
			fmt.Fprintln(fs.Output())
		}
	}

	if len(os.Args) < 2 {
		fs.Usage()
		return &AppError{
			msg:      "not enough flags",
			exitCode: 2,
		}
	}

	// Parse global flags.
	if err := fs.Parse(os.Args[1:]); err != nil {
		return &AppError{
			msg:      err.Error(),
			exitCode: 1,
		}
	}

	// Quickly bail out printing version if asked!
	if flVersion {
		printVersion()
		return nil
	}

	// Set debug mode. For now it only means enabling debug logging.
	if flDebug {
		logging.EnableDebugLogger()
	}

	// Remaining flags should be processed by subcommands.
	args := fs.Args()

	// At this point we pass on to sub-commands.
	for _, c := range subCmds {
		if args[0] == c.Name() {
			return c.Run(args[1:])
		}
	}

	// No subcommands were matched at this point, so bail out with default usage message.
	fs.Usage()
	return &AppError{
		msg:      "unsupported sub-command",
		exitCode: 2,
	}
}

func main() {
	// Enable info logger by default and early enough.
	logging.EnableInfoLogger()

	if err := root(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		switch e := err.(type) {
		case *AppError:
			os.Exit(e.ExitCode())
		default:
			os.Exit(1)
		}
	}
	os.Exit(0)
}

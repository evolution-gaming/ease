// Copyright ©2022 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// ease tool's bitrate subcommand implementation.

package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/evolution-gaming/ease/internal/analysis"
	"github.com/evolution-gaming/ease/internal/logging"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
)

// Make sure BitrateApp implements Commander interface.
var _ Commander = (*BitrateApp)(nil)

// BitrateApp is bitrate subcommand context that implements Commander interface.
type BitrateApp struct {
	// FlagSet instance
	fs *flag.FlagSet
	// Input video file path
	flInFile string
	// Plot output file
	flOutFile string
}

// CreateBitrateCommand will create Commander instance from BitrateApp.
func CreateBitrateCommand() Commander {
	longHelp := `Subcommand "bitrate" will create bitrate plot for given video file.`
	app := &BitrateApp{
		fs: flag.NewFlagSet("bitrate", flag.ContinueOnError),
	}
	app.fs.StringVar(&app.flInFile, "i", "", "Input video file (mandatory)")
	app.fs.StringVar(&app.flOutFile, "o", "", "File to save plot to")

	app.fs.Usage = func() {
		printSubCommandUsage(longHelp, app.fs)
	}
	return app
}

// Run is main entry point into BitrateApp execution.
func (a *BitrateApp) Run(args []string) error {
	if err := a.fs.Parse(args); err != nil {
		return &AppError{
			exitCode: 2,
			msg:      "usage error",
		}
	}

	if a.flInFile == "" {
		a.Help()
		return &AppError{
			exitCode: 2,
			msg:      "mandatory option -i is missing",
		}
	}

	if a.flOutFile == "" {
		base := path.Base(a.flInFile)
		base = strings.TrimSuffix(base, path.Ext(base))
		a.flOutFile = base + ".png"
	}

	logging.Infof("Output will be written to:\n\t%s\n", a.flOutFile)
	err := run(a.flInFile, a.flOutFile)
	if err != nil {
		return &AppError{
			exitCode: 1,
			msg:      err.Error(),
		}
	}

	return nil
}

func (a *BitrateApp) Name() string {
	return a.fs.Name()
}

func (a *BitrateApp) Help() {
	a.fs.Usage()
}

func run(videoFile, plotFile string) error {
	if _, err := os.Stat(videoFile); os.IsNotExist(err) {
		return fmt.Errorf("video file should exist: %w", err)
	}
	base := path.Base(videoFile)

	fs, err := analysis.GetFrameStats(videoFile)
	if err != nil {
		return fmt.Errorf("failed getting FrameStats: %w", err)
	}

	// Create a 2D slice to hold subplots. This is the state of gonum's API at this point
	// unfortunately.
	const rows, cols = 2, 1
	plots := make([][]*plot.Plot, rows)
	for i := range plots {
		plots[i] = make([]*plot.Plot, cols)
	}

	plots[0][0], err = analysis.CreateBitratePlot(fs)
	if err != nil {
		return err
	}

	plots[1][0], err = analysis.CreateFrameSizePlot(fs)
	if err != nil {
		return err
	}

	// Tweak titles and labels to have better layout and make plots less busy.
	plots[0][0].Title.Text = base + "\n\nBitrate"
	plots[0][0].X.Label.Text = ""
	plots[1][0].Title.Text = "Frame sizes"

	img := vgimg.New(vg.Centimeter*24, vg.Centimeter*14)
	dc := draw.New(img)

	t := draw.Tiles{
		Rows: rows,
		Cols: cols,
		PadY: vg.Points(10),
	}

	canvases := plot.Align(plots, t, dc)
	for j := 0; j < rows; j++ {
		for i := 0; i < cols; i++ {
			if plots[j][i] != nil {
				plots[j][i].Draw(canvases[j][i])
			}
		}
	}

	w, err := os.Create(plotFile)
	if err != nil {
		panic(err)
	}
	defer w.Close()
	png := vgimg.PngCanvas{Canvas: img}
	if _, err := png.WriteTo(w); err != nil {
		panic(err)
	}

	return nil
}

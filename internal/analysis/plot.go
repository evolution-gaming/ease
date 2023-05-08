// Copyright ©2022 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Plot generation related functionality.

package analysis

import (
	"encoding/json"
	"errors"
	"fmt"
	"image/color"
	"log"
	"math"
	"os"
	"os/exec"
	"path"
	"runtime"
	"sort"

	"github.com/evolution-gaming/ease/internal/logging"
	"gonum.org/v1/gonum/stat"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
)

var (
	defaultPlotWidth  = vg.Centimeter * 24
	defaultPlotHeight = vg.Centimeter * 7
)

// A custom color palette: color1 as base color and color2 as a darker variant.
var ColorPalette = []color.RGBA{
	// red1
	{R: 230, G: 57, B: 70, A: 255},
	// red2
	{R: 143, G: 35, B: 43, A: 255},
	// green1
	{R: 84, G: 184, B: 50, A: 255},
	// green2
	{R: 50, G: 110, B: 30, A: 255},
	// blue1
	{R: 63, G: 55, B: 201, A: 255},
	// blue2
	{R: 51, G: 45, B: 163, A: 255},
	// purple1
	{R: 86, G: 11, B: 173, A: 255},
	// purple2
	{R: 62, G: 8, B: 125, A: 255},
	// cyan1
	{R: 31, G: 180, B: 206, A: 255},
	// cyan2
	{R: 11, G: 123, B: 143, A: 255},
	// orange1
	{R: 255, G: 174, B: 0, A: 255},
	// orange2
	{R: 173, G: 118, B: 0, A: 255},
}

// CreateCDFPlot creates Cumulative Distribution Function plot for given VQM values.
func CreateCDFPlot(values []float64, name string) (*plot.Plot, error) {
	p := plot.New()
	p.X.Label.Text = name
	p.Y.Label.Text = "Probability"
	p.Y.Min = 0

	// We are going to mutate values slice, so make a copy to avoid mangling
	// underlying array and creating unexpected sideffect in caller's scope.
	lValues := make([]float64, len(values))
	copy(lValues, values)
	// Make sure values are sorted
	sort.Float64s(lValues)

	// Have to transform lValues to something that implements plotter.XYer
	// interface so it can be used later on to construct plot.
	cdfValues := make(plotter.XYs, len(lValues))
	for i, v := range lValues {
		cdfValues[i].X = v
		cdfValues[i].Y = stat.CDF(v, stat.Empirical, lValues, nil)
	}

	cdfLine, err := plotter.NewLine(cdfValues)
	if err != nil {
		return p, fmt.Errorf("CreateCDFPlot() creating new Line: %w", err)
	}
	cdfLine.Color = ColorPalette[2]

	p.Add(cdfLine, plotter.NewGrid())
	p.Add(createQuantileLines(p, lValues, 0.01, 0.05, 0.5, 0.95)...)

	return p, nil
}

// CreateHistogramPlot creates histogram plot for given VQM values.
func CreateHistogramPlot(values []float64, name string) (*plot.Plot, error) {
	p := plot.New()
	p.X.Label.Text = name
	p.Y.Label.Text = "N"

	// We are going to mutate values slice, so make a copy to avoid mangling
	// underlying array and creating unexpected sideffect in caller's scope.
	lValues := make([]float64, len(values))
	copy(lValues, values)

	// A number of bins to use for histogram.
	var bins int = 100

	// Make sure values are sorted.
	sort.Float64s(lValues)

	pHist, err := plotter.NewHist(plotter.Values(lValues), bins)
	if err != nil {
		return p, fmt.Errorf("CreateHistogramPlot() creating new histogram: %w", err)
	}
	pHist.Color = color.Transparent
	pHist.FillColor = ColorPalette[7]

	p.Add(pHist)
	p.Add(plotter.NewGrid())

	return p, nil
}

// CreateVqmPlot creates a plot for given VQM values.
//
// Since values are specified as a 1D slice - it is assumed that index into
// slice is a frame number.
func CreateVqmPlot(values []float64, name string) (*plot.Plot, error) {
	p := plot.New()
	p.X.Label.Text = "Frame #"
	p.Y.Label.Text = name

	vqmXY := make(plotter.XYs, len(values))

	for i, v := range values {
		vqmXY[i].X = float64(i)
		vqmXY[i].Y = v
	}
	vqmLine, err := plotter.NewLine(vqmXY)
	if err != nil {
		return p, fmt.Errorf("CreateVqmPlot() creating new histogram: %w", err)
	}

	vqmLine.Color = ColorPalette[0]

	p.Add(vqmLine)
	p.Add(plotter.NewGrid())

	return p, nil
}

// MultiPlotVqm will create VQM metric multi plot and save it to a file.
//
// Resulting plot will include the provided VQM metric plot, it's histogram plot
// and CDF plot all in one canvas.
func MultiPlotVqm(values []float64, metric, title, outFile string) (err error) {
	// Create a 2D slice to hold subplots. This is the sad state of gonum's API
	// at this point unfortunately.
	const rows, cols = 3, 1
	plots := make([][]*plot.Plot, rows)
	for i := range plots {
		plots[i] = make([]*plot.Plot, cols)
	}

	plots[0][0], err = CreateVqmPlot(values, metric)
	if err != nil {
		return err
	}

	plots[1][0], err = CreateHistogramPlot(values, metric)
	if err != nil {
		return err
	}

	plots[2][0], err = CreateCDFPlot(values, metric)
	if err != nil {
		return err
	}

	// Tweak titles and labels to have better layout and make plots less busy.
	plots[0][0].Title.Text = title + "\n\nPer frame " + metric
	plots[1][0].Title.Text = metric + " Histogram"
	plots[1][0].X.Label.Text = ""
	plots[2][0].Title.Text = "Cumulative Distribution Function (CDF)"

	img := vgimg.New(defaultPlotWidth, defaultPlotHeight*rows)
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

	w, err := os.Create(outFile)
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

// CreateBitratePlot creates a bitrate plot from given FrameStat slice.
func CreateBitratePlot(frameStats []FrameStat) (*plot.Plot, error) {
	p := plot.New()
	p.X.Label.Text = "Time (seconds)"
	p.Y.Label.Text = "Kbps"

	videoDuration := getDuration(frameStats)
	if videoDuration <= 0 {
		return p, errors.New("CreateBitratePlot() unexpected video duration")
	}

	var totSize uint64
	var curSecond uint64

	// Bucket count should be same as video duration in seconds.
	bSize := uint64(math.Floor(videoDuration)) + 1
	// Create buckets for all types of interesting frames.
	allFrameBuckets := make([]float64, bSize)
	iFrameBuckets := make([]float64, bSize)
	pFrameBuckets := make([]float64, bSize)

	// Aggregate frame sizes into 1 second buckets.
	minPts := frameStats[0].PtsTime
	for _, p := range frameStats {
		totSize += p.Size
		// Use normalized time e.g. deal with negative PTS.
		curSecond = uint64(math.Floor(p.PtsTime - minPts))
		// Convert frame size to Kbits.
		s := float64(p.Size*8) / 1000
		allFrameBuckets[curSecond] += s
		if p.KeyFrame {
			iFrameBuckets[curSecond] += s
		} else {
			pFrameBuckets[curSecond] += s
		}
	}

	// Prepare XYers of all frame types for plotting.
	allValues := make(plotter.XYs, len(allFrameBuckets))
	iValues := make(plotter.XYs, len(iFrameBuckets))
	pValues := make(plotter.XYs, len(pFrameBuckets))

	for i, v := range allFrameBuckets {
		allValues[i].X = float64(i)
		allValues[i].Y = v
	}

	for i, v := range iFrameBuckets {
		iValues[i].X = float64(i)
		iValues[i].Y = v
	}

	for i, v := range pFrameBuckets {
		pValues[i].X = float64(i)
		pValues[i].Y = v
	}

	// Now create all lines to be placed on plot.
	allLine, err := plotter.NewLine(allValues)
	if err != nil {
		return p, fmt.Errorf("CreateBitratePlot() creating new Line: %w", err)
	}
	allLine.Color = ColorPalette[1]
	allLine.StepStyle = plotter.PostStep
	allLine.FillColor = ColorPalette[0]

	iLine, err := plotter.NewLine(iValues)
	if err != nil {
		return p, fmt.Errorf("CreateBitratePlot() creating new I-frame Line: %w", err)
	}
	iLine.Color = ColorPalette[3]
	iLine.StepStyle = plotter.PostStep

	pLine, err := plotter.NewLine(pValues)
	if err != nil {
		return p, fmt.Errorf("CreateBitratePlot() creating new P-frame Line: %w", err)
	}
	pLine.Color = ColorPalette[5]
	pLine.StepStyle = plotter.PostStep

	// Mean and max/peak bitrate value as horizontal line.
	mean := stat.Mean(allFrameBuckets, nil)
	max := maxFloat64(allFrameBuckets)
	meanLine, meanLabel := horizontalLineWithLabel(mean, 0, float64(bSize), fmt.Sprintf("mean=%.2f", mean))
	maxLine, maxLabel := horizontalLineWithLabel(max, 0, float64(bSize), fmt.Sprintf("max=%.2f", max))

	// Tweak x and y axis limits.
	p.Y.Min = 0
	p.Y.Max = max * 1.1
	// Add ticks with period of 10 seconds.
	p.X.Tick.Marker = plot.TickerFunc(func(min, max float64) []plot.Tick {
		var t []plot.Tick
		for x := min; x <= max; x += 10 {
			t = append(t, plot.Tick{
				Value: x,
				Label: fmt.Sprintf("%.1f", x),
			})
		}
		return t
	})

	p.Add(allLine, iLine, pLine, meanLine, meanLabel, maxLine, maxLabel, plotter.NewGrid())

	p.Legend.Add("Total", allLine)
	p.Legend.Add("I-frame", iLine)
	p.Legend.Add("P-frame", pLine)
	p.Legend.Top = true
	p.Legend.XOffs = -10
	p.Legend.YOffs = -10

	return p, nil
}

func CreateFrameSizePlot(frameStats []FrameStat) (*plot.Plot, error) {
	p := plot.New()
	p.X.Label.Text = "Time (seconds)"
	p.Y.Label.Text = "KB"

	videoDuration := getDuration(frameStats)
	if videoDuration <= 0 {
		return p, errors.New("CreateFrameSizePlot() unexpected video duration")
	}

	// Prepare XYers of all frame types for plotting.
	var keyFrameSizes plotter.XYs
	var pFrameSizes plotter.XYs

	minPts := frameStats[0].PtsTime
	for _, v := range frameStats {
		xy := plotter.XY{
			// Use normalized time e.g. deal with negative PTS.
			X: float64(v.PtsTime - minPts),
			Y: float64(v.Size) / 1000,
		}

		if v.KeyFrame {
			keyFrameSizes = append(keyFrameSizes, xy)
		} else {
			pFrameSizes = append(pFrameSizes, xy)
		}
	}

	keyFrameLine, err := plotter.NewLine(keyFrameSizes)
	if err != nil {
		return p, fmt.Errorf("CreateFrameSizePlot() creating new I-frame Line: %w", err)
	}
	keyFrameLine.Color = ColorPalette[3]

	pFrameLine, err := plotter.NewLine(pFrameSizes)
	if err != nil {
		return p, fmt.Errorf("CreateFrameSizePlot() creating new P-frame Line: %w", err)
	}
	pFrameLine.Color = ColorPalette[5]

	p.Y.Min = 0
	p.X.Tick.Marker = plot.TickerFunc(func(min, max float64) []plot.Tick {
		var t []plot.Tick
		for x := min; x <= max; x += 10 {
			t = append(t, plot.Tick{
				Value: x,
				Label: fmt.Sprintf("%.1f", x),
			})
		}
		return t
	})

	p.Add(keyFrameLine, pFrameLine, plotter.NewGrid())

	return p, nil
}

// MultiPlotBitrate will create and save to file bitrate multi plot.
//
// Resulting plot will include the bitrate plot aggregated into 1 second buckets
// and frame size plot all in one canvas.
func MultiPlotBitrate(videoFile, plotFile, ffprobePath string) error {
	if _, err := os.Stat(videoFile); os.IsNotExist(err) {
		return fmt.Errorf("MultiPlotBitrate() video file should exist: %w", err)
	}
	base := path.Base(videoFile)

	fs, err := GetFrameStats(videoFile, ffprobePath)
	if err != nil {
		return fmt.Errorf("MultiPlotBitrate() failed getting FrameStats: %w", err)
	}

	// Create a 2D slice to hold subplots. This is the state of gonum's API at this point
	// unfortunately.
	const rows, cols = 2, 1
	plots := make([][]*plot.Plot, rows)
	for i := range plots {
		plots[i] = make([]*plot.Plot, cols)
	}

	plots[0][0], err = CreateBitratePlot(fs)
	if err != nil {
		return fmt.Errorf("MultiPlotBitrate() error creating bitrate plot: %w", err)
	}

	plots[1][0], err = CreateFrameSizePlot(fs)
	if err != nil {
		return fmt.Errorf("MultiPlotBitrate() error creating frame size plot: %w", err)
	}

	// Tweak titles and labels to have better layout and make plots less busy.
	plots[0][0].Title.Text = base + "\n\nBitrate"
	plots[0][0].X.Label.Text = ""
	plots[1][0].Title.Text = "Frame sizes"

	img := vgimg.New(defaultPlotWidth, defaultPlotHeight*2)
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
		return fmt.Errorf("MultiPlotBitrate() error fro os.Create(): %w", err)
	}
	defer w.Close()

	png := vgimg.PngCanvas{Canvas: img}
	if _, err := png.WriteTo(w); err != nil {
		return fmt.Errorf("MultiPlotBitrate() failed writing png file: %w", err)
	}

	return nil
}

// verticalLine is helper to create a vertical line.
func verticalLine(x, ymin, ymax float64) *plotter.Line {
	line, err := plotter.NewLine(plotter.XYs{
		{X: x, Y: ymin},
		{X: x, Y: ymax},
	})
	// Unlikely to have error here - so just panic in that case.
	if err != nil {
		log.Panic(err)
	}
	return line
}

// horizontalLine is helper to create a horizontal line.
func horizontalLine(y, xmin, xmax float64) *plotter.Line {
	line, err := plotter.NewLine(plotter.XYs{
		{X: xmin, Y: y},
		{X: xmax, Y: y},
	})
	// Unlikely to have error here - so just panic in that case.
	if err != nil {
		log.Panic(err)
	}
	return line
}

// horizontalLineWithLabel wraps horizontalLine and adds label.
func horizontalLineWithLabel(y, xMin, xMax float64, label string) (*plotter.Line, *plotter.Labels) {
	hLine := horizontalLine(y, xMin, xMax)
	hLine.Color = color.RGBA{156, 67, 162, 255}
	hLabel, _ := plotter.NewLabels(plotter.XYLabels{
		XYs: plotter.XYs{
			{X: 0, Y: y},
		},
		Labels: []string{
			label,
		},
	})
	hLabel.Offset.X = 5
	hLabel.Offset.Y = 5

	return hLine, hLabel
}

// createQuantileLines is helper to create vertical Quantile lines.
func createQuantileLines(p *plot.Plot, values []float64, quantiles ...float64) []plot.Plotter {
	var plotters []plot.Plotter
	colorCount := len(ColorPalette)
	for i, q := range quantiles {
		qVal := stat.Quantile(q, stat.Empirical, values, nil)
		qLine := verticalLine(qVal, p.Y.Min, p.Y.Max)
		qLine.LineStyle.Width = vg.Points(1)
		qLine.LineStyle.Dashes = []vg.Length{vg.Points(5), vg.Points(5)}
		// Safe index with step=2 into ColorPalette with wrap-around to avoid
		// panic in case of bounds check fails.
		qLine.Color = ColorPalette[i*5%colorCount]

		labels, _ := plotter.NewLabels(plotter.XYLabels{
			XYs: plotter.XYs{
				{X: qVal, Y: q},
			},
			Labels: []string{
				fmt.Sprintf("q(%.2f)=%.3f", q, qVal),
			},
		})
		labels.Offset.X = 5
		labels.Offset.Y = -5

		plotters = append(plotters, qLine, labels)
	}
	// Also add mean/average line.
	meanVal := stat.Mean(values, nil)
	meanLine := verticalLine(meanVal, p.Y.Min, p.Y.Max)
	meanLine.Color = ColorPalette[len(ColorPalette)-1]
	qValMean := stat.CDF(meanVal, stat.Empirical, values, nil)
	meanLabel, _ := plotter.NewLabels(plotter.XYLabels{
		XYs: plotter.XYs{
			{X: meanVal, Y: qValMean},
		},
		Labels: []string{
			fmt.Sprintf("mean=%.3f", meanVal),
		},
	})
	meanLabel.Offset.X = 5
	meanLabel.Offset.Y = -5
	plotters = append(plotters, meanLine, meanLabel)

	return plotters
}

// getDuration calculates video duration based on data from FrameStat slice.
func getDuration(fs []FrameStat) float64 {
	pts := make([]float64, 0, len(fs))
	var acc float64
	for _, v := range fs {
		acc += v.DurationTime
		pts = append(pts, v.PtsTime)
	}
	// There is no guarantee that PTS-es are in increasing order.
	sort.Float64s(pts)
	return math.Max((pts[len(pts)-1] - pts[0] + fs[0].DurationTime), acc)
}

// GetFrameStats gets per-frame stats using ffprobe.
func GetFrameStats(videoFile, ffprobePath string) ([]FrameStat, error) {
	// Although we are querying packets statistics e.g. `AVPacket` from PoV libav, still
	// for video stream it should map directly to a video frame.
	ffprobeArgs := []string{
		"-hide_banner",
		"-loglevel", "quiet",
		"-threads", fmt.Sprint(runtime.NumCPU()),
		"-select_streams", "v",
		"-show_entries",
		"packet=flags,pts_time,size,duration_time",
		"-of", "json=compact=1",
		videoFile,
	}

	cmd := exec.Command(ffprobePath, ffprobeArgs...)
	logging.Debugf("Running: %s\n", cmd)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	// Need a dummy struct for first level.
	frames := &struct {
		Packets []FrameStat
	}{}

	if err := json.Unmarshal(out, &frames); err != nil {
		return nil, err
	}

	return frames.Packets, nil
}

// FrameStat is struct with per-frame meta-data.
type FrameStat struct {
	KeyFrame     bool
	DurationTime float64
	PtsTime      float64
	Size         uint64
}

func (f *FrameStat) UnmarshalJSON(data []byte) error {
	// By convention Unmarshalers implement UnmarshalJSON([]byte("null")) as a
	// no-op.
	if string(data) == "null" {
		return nil
	}
	var ps packetStat
	if err := json.Unmarshal(data, &ps); err != nil {
		return fmt.Errorf("FrameStat.UnmarshalJSON: %w", err)
	}

	switch ps.Flags {
	case "K_":
		f.KeyFrame = true
	default:
		f.KeyFrame = false
	}
	f.DurationTime = ps.DurationTime
	f.PtsTime = ps.PtsTime
	f.Size = ps.Size

	return nil
}

// packetStat is struct with per-packet meta-data as provided by ffprobe.
type packetStat struct {
	// As reported by ffprobe flags: for key-frame it's value is "K_", we will
	// assume that all other e.g. non-key frames are P-frames although it is
	// technically incorrect since it will include B-frames as well.
	Flags        string  `json:"flags"`
	DurationTime float64 `json:"duration_time,string"`
	PtsTime      float64 `json:"pts_time,string"`
	Size         uint64  `json:"size,string"`
}

// maxFloat64 will naively find max value in slice.
func maxFloat64(values []float64) float64 {
	var max float64
	for i, v := range values {
		if i == 0 || v > max {
			max = v
		}
	}
	return max
}

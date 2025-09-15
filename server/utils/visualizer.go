package utils

import (
	"encoding/csv"
	"fmt"
	"image"
	"image/color"

	// "image/png"
	"math"
	"os"
	"strconv"

	"github.com/fogleman/gg"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

func ExportToCSV(original []float64, downsampled []float64, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	if err := writer.Write([]string{"index", "original", "downsampled"}); err != nil {
		return err
	}

	maxLen := len(original)
	if len(downsampled) > maxLen {
		maxLen = len(downsampled)
	}

	for i := 0; i < maxLen; i++ {
		var origVal, downVal string
		if i < len(original) {
			origVal = strconv.FormatFloat(original[i], 'f', -1, 64)
		} else {
			origVal = ""
		}
		if i < len(downsampled) {
			downVal = strconv.FormatFloat(downsampled[i], 'f', -1, 64)
		} else {
			downVal = ""
		}

		record := []string{strconv.Itoa(i), origVal, downVal}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}

func PlotWaveform(original []float64, downsampled []float64, filename string) error {
	p := plot.New()
	p.Title.Text = "Waveform"
	p.X.Label.Text = "Sample Index"
	p.Y.Label.Text = "Amplitude"

	// Create line plot for original
	origPts := make(plotter.XYs, len(original))
	for i := range original {
		origPts[i].X = float64(i)
		origPts[i].Y = original[i]
	}
	origLine, err := plotter.NewLine(origPts)
	if err != nil {
		return err
	}
	origLine.Color = color.RGBA{B: 255, A: 255} // Blue

	// Create line plot for downsampled
	downPts := make(plotter.XYs, len(downsampled))
	ratio := len(original) / len(downsampled)
	for i := range downsampled {
		downPts[i].X = float64(i * ratio)
		downPts[i].Y = downsampled[i]
	}
	downLine, err := plotter.NewLine(downPts)
	if err != nil {
		return err
	}
	downLine.Color = color.RGBA{R: 255, A: 255} // Red

	p.Add(origLine, downLine)
	p.Legend.Add("Original", origLine)
	p.Legend.Add("Downsampled", downLine)

	if err := p.Save(8*vg.Inch, 4*vg.Inch, filename); err != nil {
		return err
	}
	return nil
}

// PlotArrays plots one or more slices of float64 values and saves to a PNG file.
// Each dataset gets a different color/label automatically.
func PlotArrays(title, filename string, datasets ...[]float64) error {
	if len(datasets) == 0 {
		return nil
	}

	// Create plot
	p := plot.New()
	p.Title.Text = title
	p.X.Label.Text = "Index"
	p.Y.Label.Text = "Value"

	// Colors will auto-cycle
	for i, data := range datasets {
		pts := make(plotter.XYs, len(data))
		for j, v := range data {
			pts[j].X = float64(j)
			pts[j].Y = v
		}

		line, err := plotter.NewLine(pts)
		if err != nil {
			return err
		}

		line.LineStyle.Width = vg.Points(1)
		p.Add(line)
		p.Legend.Add("Series "+string(rune('A'+i)), line) // Series A, B, C...
	}

	// Save
	if err := p.Save(10*vg.Inch, 4*vg.Inch, filename); err != nil {
		return err
	}
	return nil
}

// map value [0,1] to heatmap color (black→purple→red→yellow→white)
func heatmapColor(v float64) color.RGBA {
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}

	// Simple gradient stops
	r := uint8(255 * math.Min(1, v*3))
	g := uint8(255 * math.Min(1, math.Max(0, v*3-1)))
	b := uint8(255 * math.Max(0, 1-v*3))

	return color.RGBA{R: r, G: g, B: b, A: 255}
}

// SaveSpectrogramImage saves spectrogram as grayscale or heatmap
func SaveSpectrogramWithLabels(spectrogram [][]complex128, filename string, sampleRate, hopSize int, trackDuration float64, colored bool) error {
	height := len(spectrogram[0]) / 2 // frequency bins
	width := len(spectrogram)         // time frames

	// Step 1: create spectrogram as image
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	maxVal := 0.0
	mags := make([][]float64, width)
	for t := 0; t < width; t++ {
		mags[t] = make([]float64, height)
		for f := 0; f < height; f++ {
			re := real(spectrogram[t][f])
			im := imag(spectrogram[t][f])
			mag := math.Sqrt(re*re + im*im)
			mags[t][f] = math.Log1p(mag) // log scale
			if mags[t][f] > maxVal {
				maxVal = mags[t][f]
			}
		}
	}

	for t := 0; t < width; t++ {
		for f := 0; f < height; f++ {
			val := mags[t][f] / maxVal
			if colored {
				img.Set(t, height-f-1, heatmapColor(val))
			} else {
				gray := uint8(val * 255)
				img.Set(t, height-f-1, color.Gray{Y: gray})
			}
		}
	}

	// Step 2: use gg to draw labels + embed image
	margin := 80
	canvasW := width + margin + 20
	canvasH := height + margin + 20
	dc := gg.NewContext(canvasW, canvasH)

	// Background
	dc.SetRGB(1, 1, 1)
	dc.Clear()

	// Draw spectrogram image into canvas
	dc.DrawImageAnchored(img, margin, 20, 0, 0)

	// Axes
	dc.SetRGB(0, 0, 0)
	dc.SetLineWidth(1)
	dc.DrawLine(float64(margin), 20, float64(margin), float64(height+20))
	dc.DrawLine(float64(margin), float64(height+20), float64(canvasW-20), float64(height+20))
	dc.Stroke()

	// Labels
	dc.SetRGB(0, 0, 0)
	if err := dc.LoadFontFace("/System/Library/Fonts/Supplemental/Arial.ttf", 12); err != nil {
		return err
	}

	// X-axis (time)
	totalTime := trackDuration
	fmt.Println("total time in image: ", totalTime)
	for sec := 0; sec <= int(totalTime); sec++ {
		time := float64(sec)
		x := margin + int(float64(width)*time/totalTime)
		dc.DrawStringAnchored(fmt.Sprintf("%ds", sec), float64(x), float64(height+40), 0.5, 0.5)
		dc.DrawLine(float64(x), float64(height+20), float64(x), float64(height+25))
		dc.Stroke()
	}
	dc.DrawStringAnchored("Time (s)", float64(margin+width/2), float64(height+65), 0.5, 0.5)

	// Y-axis (frequency)
	maxFreq := sampleRate / 2
	for i := 0; i <= 10; i++ {
		y := 20 + height - int(float64(height)*float64(i)/10.0)
		freq := float64(maxFreq) * float64(i) / 10.0
		dc.DrawStringAnchored(fmt.Sprintf("%.0f Hz", freq), float64(margin-10), float64(y), 1, 0.5)
		dc.DrawLine(float64(margin-5), float64(y), float64(margin), float64(y))
		dc.Stroke()
	}
	dc.DrawStringAnchored("Frequency (Hz)", 20, float64(height/2+20), 0.5, 0.5)

	// Step 3: Save
	return dc.SavePNG(filename)
}

// SaveConstellationMap saves a visualization of peaks (time, frequency) as an image.
// peaks is a slice of [2]float64 where index 0 is time (in seconds) and index 1 is frequency (Hz).
// sampleRate is used to scale frequency axis, maxTime is the duration in seconds to scale time axis.
// The output image has labeled axes similar to SaveSpectrogramWithLabels.
func SaveConstellationMap(peaks [][2]float64, filename string, sampleRate int, maxTime float64) error {
	// Define canvas and margin sizes
	margin := 80
	canvasW := 800 + margin + 20
	canvasH := 400 + margin + 20

	// Create a new drawing context
	dc := gg.NewContext(canvasW, canvasH)

	// Background: white
	dc.SetRGB(1, 1, 1)
	dc.Clear()

	// Define plotting area dimensions
	plotWidth := 800
	plotHeight := 400

	// Draw axes lines
	dc.SetRGB(0, 0, 0)
	dc.SetLineWidth(1)
	// Y axis
	dc.DrawLine(float64(margin), 20, float64(margin), float64(plotHeight+20))
	// X axis
	dc.DrawLine(float64(margin), float64(plotHeight+20), float64(margin+plotWidth), float64(plotHeight+20))
	dc.Stroke()

	// Load font for labels
	if err := dc.LoadFontFace("/System/Library/Fonts/Supplemental/Arial.ttf", 12); err != nil {
		return err
	}

	// Draw X-axis labels (time in seconds)
	for sec := 0; sec <= int(maxTime); sec++ {
		x := margin + int(float64(plotWidth)*float64(sec)/maxTime)
		dc.DrawStringAnchored(fmt.Sprintf("%ds", sec), float64(x), float64(plotHeight+40), 0.5, 0.5)
		dc.DrawLine(float64(x), float64(plotHeight+20), float64(x), float64(plotHeight+25))
		dc.Stroke()
	}
	dc.DrawStringAnchored("Time (s)", float64(margin+plotWidth/2), float64(plotHeight+65), 0.5, 0.5)

	// Draw Y-axis labels (frequency in Hz)
	maxFreq := sampleRate / 2
	for i := 0; i <= 10; i++ {
		y := 20 + plotHeight - int(float64(plotHeight)*float64(i)/10.0)
		freq := float64(maxFreq) * float64(i) / 10.0
		dc.DrawStringAnchored(fmt.Sprintf("%.0f Hz", freq), float64(margin-10), float64(y), 1, 0.5)
		dc.DrawLine(float64(margin-5), float64(y), float64(margin), float64(y))
		dc.Stroke()
	}
	dc.DrawStringAnchored("Frequency (Hz)", 20, float64(plotHeight/2+20), 0.5, 0.5)

	// Draw peaks as small red circles
	dc.SetRGB(1, 0, 0) // red color
	radius := 2.0

	for _, peak := range peaks {
		timeVal := peak[0]
		freqVal := peak[1]

		// Skip peaks outside the plotting range
		if timeVal < 0 || timeVal > maxTime {
			continue
		}
		if freqVal < 0 || freqVal > float64(maxFreq) {
			continue
		}

		// Map time and frequency to canvas coordinates
		x := float64(margin) + (float64(plotWidth) * timeVal / maxTime)
		y := 20 + float64(plotHeight) - (float64(plotHeight) * freqVal / float64(maxFreq))

		// Draw circle
		dc.DrawCircle(x, y, radius)
		dc.Fill()
	}

	// Save image to file
	return dc.SavePNG(filename)
}

func VerifySpectrogramCompleteness(spectrogram [][]complex128,
	originalAudioSamples []float64, sampleRate, hopSize int) {

	// Calculate expected vs actual
	expectedFrames := (len(originalAudioSamples)-2048)/hopSize + 1
	actualFrames := len(spectrogram)

	expectedDuration := float64(len(originalAudioSamples)) / float64(sampleRate)
	actualDuration := float64(actualFrames*hopSize) / float64(sampleRate)

	fmt.Printf("=== Spectrogram Completeness Check ===\n")
	fmt.Printf("Expected frames: %d | Actual frames: %d\n", expectedFrames, actualFrames)
	fmt.Printf("Expected duration: %.2fs | Actual duration: %.2fs\n",
		expectedDuration, actualDuration)

	completeness := float64(actualFrames) / float64(expectedFrames) * 100
	fmt.Printf("Coverage: %.1f%%\n", completeness)

	if completeness < 95 {
		fmt.Printf("⚠️  WARNING: Spectrogram covers only %.1f%% of the track!\n", completeness)
	} else {
		fmt.Printf("✅ Spectrogram covers the complete track\n")
	}
}

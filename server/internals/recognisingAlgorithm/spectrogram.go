package recognisingalgorithm

import (
	"errors"
	"fmt"
	"math"
)

const (
	DSPratio    = 4
	freqBinSize = 1024
	hopSize     = freqBinSize / 32
)

func Spectrogram(sample []float64, sampleRate int) ([][]complex128, error) {
	fmt.Printf("duration of the track is : %v\n", len(sample)/sampleRate)

	// Downsample first
	downSampled, err := DownSampleProper(sampleRate, sampleRate/DSPratio, sample)
	if err != nil {
		return nil, errors.New("error downsampling the audio sample")
	}

	// Frame params
	frameSize := freqBinSize // must be power of 2 for FFT
	hop := hopSize
	window := make([]float64, frameSize)
	for i := range window {
		window[i] = 0.54 - 0.46*math.Cos(2*math.Pi*float64(i)/float64(frameSize-1)) // Hamming
	}

	// Number of frames
	numFrames := 1 + (len(downSampled)-frameSize)/hop
	spectrogram := make([][]complex128, numFrames)

	for i := 0; i < numFrames; i++ {
		start := i * hop
		frame := make([]float64, frameSize)
		copy(frame, downSampled[start:start+frameSize])

		// Apply window
		for j := range frame {
			frame[j] *= window[j]
		}

		// FFT
		spectrogram[i] = fftRealToComplex(frame)
	}

	fmt.Println("Spectrogram frames:", len(spectrogram))
	return spectrogram, nil
}

// Low-pass FIR filter generator (windowed sinc)
func lowPassFIR(cutoff, sampleRate float64, taps int) []float64 {
	h := make([]float64, taps)
	normCutoff := cutoff / sampleRate // normalized cutoff (0..0.5)

	for i := 0; i < taps; i++ {
		m := float64(i - taps/2)
		if m == 0 {
			h[i] = 2 * normCutoff
		} else {
			h[i] = math.Sin(2*math.Pi*normCutoff*m) / (math.Pi * m)
		}
		// Hann window to reduce spectral leakage
		h[i] *= 0.5 * (1 - math.Cos(2*math.Pi*float64(i)/float64(taps-1)))
	}
	return h
}

// Convolve input with FIR filter
func applyFIR(input, coeffs []float64) []float64 {
	out := make([]float64, len(input))
	for i := range input {
		sum := 0.0
		for j := range coeffs {
			if i-j >= 0 {
				sum += coeffs[j] * input[i-j]
			}
		}
		out[i] = sum
	}
	return out
}

// Proper downsampling
func DownSampleProper(originalSampleRate, targetSampleRate int, input []float64) ([]float64, error) {
	if targetSampleRate <= 0 || originalSampleRate <= 0 {
		return nil, errors.New("sample rates must be positive")
	}
	if targetSampleRate > originalSampleRate {
		return nil, errors.New("target sample rate must be <= original sample rate")
	}

	ratio := originalSampleRate / targetSampleRate
	cutoff := float64(targetSampleRate) / 2 // Nyquist of target SR

	// Design and apply low-pass filter
	filter := lowPassFIR(cutoff, float64(originalSampleRate), 64)
	filtered := applyFIR(input, filter)

	// Decimate
	output := make([]float64, 0, len(filtered)/ratio+1)
	for i := 0; i < len(filtered); i += ratio {
		output = append(output, filtered[i])
	}
	return output, nil
}

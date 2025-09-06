package recognisingalgorithm

import (
	"errors"
	"math"
)

const (
	DSPratio     = 4
	maxFrequency = 5000
	freqBinSize  = 1024
	hopSize      = freqBinSize / 32
)

func Spectrogram(sample []float64, sampleRate int) ([][]complex128, error) {
	// Compute the spectrogram using Short-Time Fourier Transform (STFT))
	downSampled, err := DownSampleProper(sampleRate, sampleRate/DSPratio, sample)
	// utils.PlotWaveform(sample, downSampled, "waveform.png")
	if err != nil {
		return nil, errors.New("error downsamoling the audio sample.")
	}
	windowNum := len(downSampled) / (freqBinSize - hopSize)
	spectrogram := make([][]complex128, windowNum)
	window := make([]float64, freqBinSize)
	for i := range window {
		window[i] = 0.54 - 0.46*math.Cos(2*math.Pi*float64(i)/float64(freqBinSize-1)) // Hamming window
	}
	// apply STFT
	for i := 0; i < windowNum; i++ {
		start := i * hopSize
		end := start + freqBinSize
		if end > len(downSampled) {
			end = len(downSampled)
		}
		bin := make([]float64, freqBinSize)
		copy(bin, downSampled[start:end])

		for j := range window {
			bin[j] *= window[j]
		}
		spectrogram[i] = fftRealToComplex(bin)
	}
	return spectrogram, nil
}

//

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

// func LowPassFilter(cutOffFreq, sampleRate float64, sampleInput []float64) []float64 {
// 	rc := 1.0 / (2 * math.Pi * cutOffFreq)
// 	dt := 1.0 / sampleRate
// 	alpha := dt / (rc + dt)

// 	sampleOutput := make([]float64, len(sampleInput))
// 	sampleOutput[0] = sampleInput[0] // Initialize the first output sample
// 	previousOutput := 0.0
// 	// The standard low-pass filter formula is:
// 	// y[i] = y[i-1] + alpha * (x[i] - y[i-1])
// 	// where:
// 	// y[i] is the current output sample
// 	// y[i-1] is the previous output sample
// 	// x[i] is the current input sample
// 	// alpha is the smoothing factor, calculated as:
// 	// alpha = dt / (rc + dt)
// 	// rc = 1 / (2 * π * cutOffFreq)
// 	// dt = 1 / sampleRate
// 	// However,
// 	// here we are using a slightly different formula
// 	// y[i] = (x[i] * alpha) + (y[i-1] * (1 - alpha))
// 	// which is mathematically equivalent to the above formula
// 	// but avoids the subtraction operation which can introduce numerical instability
// 	// especially when x[i] and y[i-1] are very close in value
// 	// this is a common technique used in digital signal processing to improve stability

// 	// using the above formula
// 	for i, v := range sampleInput {
// 		if i == 0 {
// 			sampleOutput[i] = v * alpha
// 		} else {
// 			sampleOutput[i] = (v * alpha) + (previousOutput * (1 - alpha))
// 		}
// 		previousOutput = sampleOutput[i]
// 	}
// 	return sampleOutput
// }

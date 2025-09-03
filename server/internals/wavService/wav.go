package wavservice

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Pritam-deb/echo-sense/utils"
)

type WavHeader struct {
	ChunkID       [4]byte // "RIFF"
	ChunkSize     uint32
	Format        [4]byte // "WAVE"
	Subchunk1ID   [4]byte // "fmt "
	Subchunk1Size uint32  // 16 for PCM
	AudioFormat   uint16  // PCM = 1
	NumChannels   uint16
	SampleRate    uint32
	ByteRate      uint32
	BlockAlign    uint16
	BitsPerSample uint16
	Subchunk2ID   [4]byte // "data"
	Subchunk2Size uint32
}

type WavInformation struct {
	NumChannels   uint16
	SampleRate    uint32
	BitsPerSample uint16
	Duration      float64 // in seconds
	Data          []byte
}

func ConvertToWav(inputFilePath string, channels int) (wavFilePath string, err error) {
	_, err = os.Stat(inputFilePath)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("input file does not exist: %s", inputFilePath)
	}

	if channels < 1 || channels > 2 {
		channels = 1 // default to mono if invalid channel count
	}

	fileExt := filepath.Ext(inputFilePath)
	outputFile := strings.TrimSuffix(inputFilePath, fileExt) + ".wav"
	tmpFile := filepath.Join(filepath.Dir(outputFile), "tmp_"+filepath.Base(outputFile))
	defer os.Remove(tmpFile) // clean up temp file if exists

	// Construct ffmpeg command
	cmdArgs := []string{"-y", "-i", inputFilePath, "-c", "pcm_s16le", "-ar", "44100", "-ac", fmt.Sprint(channels), tmpFile}
	cmd := exec.Command("ffmpeg", cmdArgs...)
	output, err := cmd.CombinedOutput() // run command
	if err != nil {
		return "", fmt.Errorf("ffmpeg error: %v, output: %s", err, string(output))
	}
	err = utils.MoveFile(tmpFile, outputFile) // rename temp file to final output
	if err != nil {
		return "", fmt.Errorf("failed to rename temp file: %v", err)
	}
	fmt.Println("WAV file created at:", outputFile)
	return outputFile, nil
}

func ReadWavFile(fileName string) (*WavInformation, error) {
	data, err := os.ReadFile(fileName)
	if err != nil {
		return nil, err
	}
	if len(data) < 44 {
		return nil, fmt.Errorf("file too small to be a valid WAV file")
	}
	var header WavHeader
	err = binary.Read(bytes.NewReader(data[:44]), binary.LittleEndian, &header)
	if err != nil {
		return nil, err
	}
	fmt.Printf("WAV Header: %+v\n", header)

	if string(header.ChunkID[:]) != "RIFF" || string(header.Format[:]) != "WAVE" || string(header.Subchunk1ID[:]) != "fmt " || header.AudioFormat != 1 {
		return nil, fmt.Errorf("invalid WAV file format")
	}

	info := &WavInformation{
		NumChannels:   header.NumChannels,
		SampleRate:    header.SampleRate,
		BitsPerSample: header.BitsPerSample,
		Data:          data[44:],
	}
	fmt.Printf("WAV Info - Channels: %d, SampleRate: %d, BitsPerSample: %d, Duration: %.2f seconds\n", info.NumChannels, info.SampleRate, info.BitsPerSample, info.Duration)
	if header.BitsPerSample != 16 {
		return nil, fmt.Errorf("unsupported BitsPerSample: %d, only 16 is supported", header.BitsPerSample)
	}
	info.Duration = float64(len(info.Data)) / float64(int(header.NumChannels)*2*int(header.SampleRate))
	return info, nil
}

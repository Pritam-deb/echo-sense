package wavservice

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Pritam-deb/echo-sense/utils"
)

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

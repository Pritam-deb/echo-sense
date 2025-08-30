package utils

import "os"

func CreateDirIfNotExist(path string) error {
	err := os.MkdirAll(path, 0750)
	if err != nil {
		return err
	}
	return nil
}

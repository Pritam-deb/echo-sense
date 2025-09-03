package utils

import (
	"log/slog"
	"os"
)

func CreateDirIfNotExist(path string) error {
	err := os.MkdirAll(path, 0750)
	if err != nil {
		return err
	}
	return nil
}

func replaceAttribute(groups []string, a slog.Attr) slog.Attr {
	if a.Key == slog.TimeKey {
		a.Value = slog.StringValue(a.Value.Time().Format("2006-01-02 15:04:05"))
	}
	return a
}

func GetLogger() *slog.Logger {
	logger := slog.New(slog.NewJSONHandler(
		os.Stdout, &slog.HandlerOptions{ReplaceAttr: replaceAttribute},
	))
	return logger
}

func GetEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func MoveFile(src, dst string) error {
	err := os.Rename(src, dst)
	if err != nil {
		return err
	}
	return nil
}

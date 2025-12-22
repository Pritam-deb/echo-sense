package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/Pritam-deb/echo-sense/handlers"
	"github.com/Pritam-deb/echo-sense/utils"

	"github.com/Pritam-deb/echo-sense/db"
	"github.com/joho/godotenv"
)

func init() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("No .env file found")
	}
}

func main() {
	// Entry point of the server application
	err := utils.CreateDirIfNotExist("temporary_files")
	if err != nil {
		logger := utils.GetLogger()
		ctx := context.Background()
		logger.ErrorContext(ctx, "Failed to create temp directory", slog.Any("error", err))
	}

	if len(os.Args) < 2 {
		fmt.Println("Expected at least one arguement of the following: 'find', 'download', 'save', 'erase', 'serve'")
		fmt.Println("Example: go run main.go <arguement>")

		os.Exit(1)
	}

	err = utils.CreateDirIfNotExist(handlers.SONGS_DIR)
	if err != nil {
		logger := utils.GetLogger()
		ctx := context.Background()
		logger.ErrorContext(ctx, "Failed to create directory for songs", slog.Any("error", err))
	}

	db.Connect()

	switch os.Args[1] {
	case "download":
		if len(os.Args) < 3 {
			fmt.Println("Expected URL after 'download'.")
			fmt.Println("Example: go run main.go download <spotify_url>")
			os.Exit(1)
		}
		url := os.Args[2]
		handlers.Download(url)

	case "search":
		if len(os.Args) < 3 {
			fmt.Println("Expected path after 'search'.")
			fmt.Println("Example: go run main.go search \"Artist - Title\"")
			os.Exit(1)
		}
		songTitle := os.Args[2]
		handlers.Search(songTitle)

	case "serve":
		serverCmd := flag.NewFlagSet("serve", flag.ExitOnError)
		protocol := serverCmd.String("protocol", "http", "Protocol to run the server on (http or https)")
		port := serverCmd.String("port", "8080", "Port to run the server on")
		serverCmd.Parse(os.Args[2:])
		handlers.Serve(*protocol, *port)

	default:
		fmt.Println("Unknown command. Expected one of: 'search', 'download', 'serve'")
		fmt.Println("Example: go run main.go <arguement>")
		os.Exit(1)
	}

}

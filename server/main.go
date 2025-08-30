package main

import "github.com/Pritam-deb/echo-sense/utils"

func main() {
	// Entry point of the server application
	err := utils.CreateDirIfNotExist("temporary_files")
	if err != nil {
		panic(err)
	}
}

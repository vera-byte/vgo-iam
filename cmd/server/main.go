package main

import (
	"log"

	cmd "github.com/vera-byte/vgo-iam/cmd"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

package main

import (
	"log"

	"github.com/amiraminb/coinwarrior/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

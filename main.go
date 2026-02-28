package main

import (
	"log"

	"clawproxy/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

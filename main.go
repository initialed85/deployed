package main

import (
	"log"

	"github.com/initialed85/deployed/pkg/app"
)

func main() {
	err := app.App()
	if err != nil {
		log.Fatal(err)
	}
}

//go:generate ./version.sh

package main

import (
	"log"

	"github.com/initialed85/deployed/pkg/app"

	_ "embed"
)

//go:embed VERSION
var version string

func main() {
	err := app.App(version)
	if err != nil {
		log.Fatal(err)
	}
}

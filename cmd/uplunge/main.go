// Command uplunge runs the game.
package main

import (
	"flag"
	"log"

	"github.com/tongmon/uplunge/internal/app"
)

func main() {
	scale := flag.Int("scale", 2, "integer window scale")
	flag.Parse()

	if err := app.Run(app.Config{Scale: *scale}); err != nil {
		log.Fatal(err)
	}
}

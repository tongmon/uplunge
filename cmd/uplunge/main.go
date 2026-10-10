// Command uplunge runs the game.
package main

import (
	"flag"
	"log"

	"github.com/tongmon/uplunge/internal/app"
)

func main() {
	scale := flag.Int("scale", 2, "integer window scale")
	tuningPath := flag.String("tuning", "data/tuning.json", "tuning JSON file")
	flag.Parse()

	if err := app.Run(app.Config{Scale: *scale, TuningPath: *tuningPath}); err != nil {
		log.Fatal(err)
	}
}

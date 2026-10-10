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
	chunksPath := flag.String("chunks", "assets/chunks/chunks.ldtk", "LDtk project with the level chunks")
	chunk := flag.String("chunk", "", "chunk to play in (default: the first one)")
	flag.Parse()

	cfg := app.Config{
		Scale:      *scale,
		TuningPath: *tuningPath,
		ChunksPath: *chunksPath,
		Chunk:      *chunk,
	}
	if err := app.Run(cfg); err != nil {
		log.Fatal(err)
	}
}

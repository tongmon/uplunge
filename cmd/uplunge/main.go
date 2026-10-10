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
	chunk := flag.String("chunk", "", "chunk to play in (default: the replay's chunk, else the first one)")
	replayPath := flag.String("replay", "", "play back inputs from this replay file, then exit")
	recordPath := flag.String("record", "", "save this run's inputs to this replay file on exit")
	flag.Parse()

	cfg := app.Config{
		Scale:      *scale,
		TuningPath: *tuningPath,
		ChunksPath: *chunksPath,
		Chunk:      *chunk,
		ReplayPath: *replayPath,
		RecordPath: *recordPath,
	}
	if err := app.Run(cfg); err != nil {
		log.Fatal(err)
	}
}

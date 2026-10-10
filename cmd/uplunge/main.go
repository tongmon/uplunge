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
	chunk := flag.String("chunk", "", "play from the top of this one chunk instead of climbing a tower")
	seed := flag.Uint64("seed", 0, "seed of the tower or lab to climb (default: a new one from the clock, logged)")
	lab := flag.Bool("lab", false, "climb a lab shaft with enemies every tuning lab.spacing px, to try out the ratio")
	replayPath := flag.String("replay", "", "play back inputs from this replay file, then exit")
	recordPath := flag.String("record", "", "save this run's inputs to this replay file on exit")
	shots := flag.String("shots", "", "comma-separated ticks to save as PNGs, then exit (e.g. 0,30,120)")
	shotsDir := flag.String("shots-dir", "out/shots", "directory for -shots PNGs")
	reload := flag.Bool("reload", false, "reload the -tuning file while running whenever it changes")
	flag.Parse()

	var shotTicks []uint64
	if *shots != "" {
		var err error
		if shotTicks, err = app.ParseShotTicks(*shots); err != nil {
			log.Fatal(err)
		}
	}

	var seedPtr *uint64
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "seed" {
			seedPtr = seed
		}
	})

	cfg := app.Config{
		Scale:      *scale,
		TuningPath: *tuningPath,
		ChunksPath: *chunksPath,
		Chunk:      *chunk,
		Seed:       seedPtr,
		Lab:        *lab,
		ReplayPath: *replayPath,
		RecordPath: *recordPath,
		ShotTicks:  shotTicks,
		ShotsDir:   *shotsDir,
		Reload:     *reload,
	}
	if err := app.Run(cfg); err != nil {
		log.Fatal(err)
	}
}

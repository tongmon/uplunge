// Package app wires the simulation to Ebitengine: it owns the window, drives
// the fixed-step loop, and draws the world.
package app

import (
	"errors"
	"fmt"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"github.com/tongmon/uplunge/internal/input"
	"github.com/tongmon/uplunge/internal/level"
	"github.com/tongmon/uplunge/internal/render"
	"github.com/tongmon/uplunge/internal/replay"
	"github.com/tongmon/uplunge/internal/sim"
	"github.com/tongmon/uplunge/internal/tuning"
)

// Logical screen size in pixels: 13 tiles of 16 px across, fixed height so
// every device sees the same amount of the tower.
const (
	ScreenWidth  = 208
	ScreenHeight = 360
)

// Config holds startup options.
type Config struct {
	// Scale is the integer window scale relative to the logical screen.
	Scale int
	// TuningPath is the tuning JSON file to load.
	TuningPath string
	// ChunksPath is the LDtk project holding the level chunks.
	ChunksPath string
	// Chunk names the chunk to play in. Empty means the replay's chunk, or
	// else the first chunk.
	Chunk string
	// ReplayPath, if set, plays back recorded inputs instead of reading the
	// keyboard and exits when they run out.
	ReplayPath string
	// RecordPath, if set, saves every step's input there when the game exits.
	RecordPath string
}

// Run opens the window and blocks until the game exits.
func Run(cfg Config) error {
	if cfg.Scale < 1 {
		return fmt.Errorf("app: scale must be at least 1, got %d", cfg.Scale)
	}
	tun, err := tuning.Load(cfg.TuningPath)
	if err != nil {
		return err
	}
	chunks, err := level.LoadLDtk(cfg.ChunksPath)
	if err != nil {
		return err
	}

	g := &game{recording: cfg.RecordPath != ""}
	chunk := cfg.Chunk
	var rec replay.Replay
	if cfg.ReplayPath != "" {
		if rec, err = replay.Load(cfg.ReplayPath); err != nil {
			return err
		}
		if chunk != "" && chunk != rec.Chunk {
			return fmt.Errorf("app: -chunk %q does not match the replay's chunk %q", chunk, rec.Chunk)
		}
		chunk = rec.Chunk
		g.playback = rec.Inputs
		g.replaying = true
	}
	if chunk == "" {
		chunk = chunks[0].Name
	}
	m, err := level.FindChunk(chunks, chunk)
	if err != nil {
		return err
	}
	if g.replaying {
		for _, msg := range rec.Mismatches(tun.Fingerprint(), m.Fingerprint()) {
			log.Printf("warning: %s: %s; playback may diverge", cfg.ReplayPath, msg)
		}
	}
	if g.world, err = sim.NewWorldInChunk(tun, m); err != nil {
		return err
	}

	ebiten.SetWindowTitle("uplunge")
	ebiten.SetWindowSize(ScreenWidth*cfg.Scale, ScreenHeight*cfg.Scale)
	ebiten.SetTPS(sim.Hz)
	ebiten.SetScreenFilterEnabled(false)
	runErr := ebiten.RunGame(g)
	if g.recording {
		out := replay.Replay{Chunk: chunk, Tuning: tun.Fingerprint(), Map: m.Fingerprint(), Inputs: g.recorded}
		if err := replay.Save(cfg.RecordPath, out); err != nil {
			return errors.Join(runErr, err)
		}
	}
	return runErr
}

type game struct {
	world *sim.World

	replaying bool
	playback  []sim.Input
	recording bool
	recorded  []sim.Input
}

// Update runs exactly one simulation step. Ebitengine calls it sim.Hz times
// per second and catches up with extra calls when a frame runs long.
func (g *game) Update() error {
	var in sim.Input
	if g.replaying {
		if g.world.Tick >= uint64(len(g.playback)) {
			return ebiten.Termination
		}
		in = g.playback[g.world.Tick]
	} else {
		in = input.Read()
	}
	if g.recording {
		g.recorded = append(g.recorded, in)
	}
	g.world.Step(in)
	return nil
}

func (g *game) Draw(screen *ebiten.Image) {
	render.World(screen, g.world)
	p := g.world.Player
	ebitenutil.DebugPrint(screen, fmt.Sprintf("tick %d  fps %.0f\nx %d y %d\nvx %.0f vy %.0f",
		g.world.Tick, ebiten.ActualFPS(), p.Body.X, p.Body.Y, p.VX, p.VY))
}

func (g *game) Layout(int, int) (int, int) {
	return ScreenWidth, ScreenHeight
}

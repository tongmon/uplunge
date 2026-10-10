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
	// ShotTicks, if set, saves the world as a PNG in ShotsDir at each of these
	// ticks (ascending) and exits after the last one.
	ShotTicks []uint64
	ShotsDir  string
	// Reload, if set, reloads TuningPath while the game runs whenever the file
	// changes.
	Reload bool
}

// reloadPollSteps is how often, in steps, Reload checks the tuning file.
const reloadPollSteps = sim.Hz / 2

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
	if cfg.Reload {
		if g.recording {
			return fmt.Errorf("app: -reload cannot be combined with -record: the replay would not reproduce the run")
		}
		g.reloadPath = cfg.TuningPath
	}
	if len(cfg.ShotTicks) > 0 {
		last := cfg.ShotTicks[len(cfg.ShotTicks)-1]
		if g.replaying && last > uint64(len(g.playback)) {
			return fmt.Errorf("app: shot tick %d is past the replay's end at tick %d", last, len(g.playback))
		}
		if g.shots, err = newShooter(cfg.ShotsDir, cfg.ShotTicks); err != nil {
			return err
		}
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

	shots *shooter

	reloadPath    string // empty unless -reload
	reloadWait    int
	lastReloadErr string
}

// Update runs exactly one simulation step. Ebitengine calls it sim.Hz times
// per second and catches up with extra calls when a frame runs long.
func (g *game) Update() error {
	if g.shots != nil {
		if err := g.shots.maybeShoot(g); err != nil {
			return err
		}
		if g.shots.done() {
			return ebiten.Termination
		}
	}
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
	if g.reloadPath != "" {
		g.pollTuning()
	}
	g.world.Step(in)
	return nil
}

// pollTuning applies tuning file edits between steps. The same error is
// logged only once, so a broken file or a blocked resize does not flood the log
// while it is retried.
func (g *game) pollTuning() {
	if g.reloadWait--; g.reloadWait > 0 {
		return
	}
	g.reloadWait = reloadPollSteps

	applied, err := reloadTuning(g.world, g.reloadPath)
	if err != nil {
		if msg := err.Error(); msg != g.lastReloadErr {
			log.Printf("tuning reload at tick %d: %v", g.world.Tick, err)
			g.lastReloadErr = msg
		}
		return
	}
	g.lastReloadErr = ""
	if applied {
		log.Printf("tuning reloaded at tick %d", g.world.Tick)
	}
}

// reloadTuning reads path and, if its values differ from the ones w steps
// with, applies them. The file is re-read and compared every time instead of
// trusting its modification time, so a failed read is retried on the next
// call, an edit that keeps the size and time is still seen, and an edit saved
// while the game was starting is not missed. applied reports whether w
// changed; err is set when the file could not be loaded (w unchanged) or when
// the new player size did not fit (other values applied).
func reloadTuning(w *sim.World, path string) (applied bool, err error) {
	t, err := tuning.Load(path)
	if err != nil {
		return false, fmt.Errorf("keeping the current values: %w", err)
	}
	if t.Fingerprint() == w.Tuning().Fingerprint() {
		return false, nil
	}
	return true, w.SetTuning(t)
}

func (g *game) Draw(screen *ebiten.Image) {
	render.World(screen, g.world)
	p := g.world.Player
	ebitenutil.DebugPrint(screen, fmt.Sprintf("tick %d  fps %.0f\nx %d y %d\nvx %.0f vy %.0f\nfuel %d/%d",
		g.world.Tick, ebiten.ActualFPS(), p.Body.X, p.Body.Y, p.VX, p.VY, p.Fuel, g.world.Tuning().Gun.Magazine))
}

func (g *game) Layout(int, int) (int, int) {
	return ScreenWidth, ScreenHeight
}

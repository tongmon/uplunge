// Package app wires the simulation to Ebitengine: it owns the window, drives
// the fixed-step loop, and draws the world.
package app

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"github.com/tongmon/uplunge/internal/input"
	"github.com/tongmon/uplunge/internal/level"
	"github.com/tongmon/uplunge/internal/render"
	"github.com/tongmon/uplunge/internal/replay"
	"github.com/tongmon/uplunge/internal/sim"
	"github.com/tongmon/uplunge/internal/tuning"
)

// Logical screen size in pixels: the simulation's view.
const (
	ScreenWidth  = sim.ViewWidth
	ScreenHeight = sim.ViewHeight
)

// Config holds startup options.
type Config struct {
	// Scale is the integer window scale relative to the logical screen.
	Scale int
	// TuningPath is the tuning JSON file to load.
	TuningPath string
	// ChunksPath is the LDtk project holding the level chunks.
	ChunksPath string
	// Chunk, if set, plays from the top of this one chunk instead of
	// climbing a tower.
	Chunk string
	// Seed, if set, is the seed of the tower to climb. Otherwise a new tower
	// is stacked from the clock, and its seed is logged.
	Seed *uint64
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

	g := &game{recording: cfg.RecordPath != "", chunks: chunks}
	start, err := startOf(cfg)
	if err != nil {
		return err
	}
	g.newSeeds = start.Tower && cfg.Seed == nil && cfg.ReplayPath == ""
	if cfg.ReplayPath != "" {
		g.playback = start.Inputs
		g.replaying = true
	} else if start.Tower && cfg.Seed == nil {
		log.Printf("climbing tower %d (replay it with -seed %d)", start.Seed, start.Seed)
	}
	if err := g.begin(start, tun); err != nil {
		return err
	}
	m := g.startMap
	if g.replaying {
		for _, msg := range start.Mismatches(tun.Fingerprint(), m.Fingerprint()) {
			log.Printf("warning: %s: %s; playback may diverge", cfg.ReplayPath, msg)
		}
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
		out := replay.Replay{Tower: g.start.Tower, Seed: g.start.Seed, Chunk: g.start.Chunk,
			Tuning: tun.Fingerprint(), Map: g.startMap.Fingerprint(), Inputs: g.recorded}
		if err := replay.Save(cfg.RecordPath, out); err != nil {
			return errors.Join(runErr, err)
		}
	}
	return runErr
}

// startOf decides where the run starts: where the replay started, else the
// -chunk chunk, else a tower with the -seed seed or one from the clock. For a
// replay it also holds the inputs.
func startOf(cfg Config) (replay.Replay, error) {
	if cfg.ReplayPath != "" {
		r, err := replay.Load(cfg.ReplayPath)
		if err != nil {
			return r, err
		}
		switch {
		case cfg.Chunk != "" && (r.Tower || cfg.Chunk != r.Chunk):
			return r, fmt.Errorf("app: -chunk %q does not match the replay, which starts in %s", cfg.Chunk, r.Where())
		case cfg.Seed != nil && (!r.Tower || *cfg.Seed != r.Seed):
			return r, fmt.Errorf("app: -seed %d does not match the replay, which starts in %s", *cfg.Seed, r.Where())
		}
		return r, nil
	}
	switch {
	case cfg.Chunk != "" && cfg.Seed != nil:
		return replay.Replay{}, fmt.Errorf("app: -seed picks a tower and -chunk a single chunk; give one")
	case cfg.Chunk != "":
		return replay.Replay{Chunk: cfg.Chunk}, nil
	case cfg.Seed != nil:
		return replay.Replay{Tower: true, Seed: *cfg.Seed}, nil
	}
	return replay.Replay{Tower: true, Seed: uint64(time.Now().UnixNano())}, nil
}

type game struct {
	world *sim.World
	// start and startMap are where the current run started.
	start    replay.Replay
	startMap *level.TileMap
	chunks   []level.Chunk
	// newSeeds makes each restart climb a new tower from the clock, unless
	// -seed fixed it.
	newSeeds bool

	replaying bool
	playback  []sim.Input
	recording bool
	recorded  []sim.Input

	shots *shooter

	reloadPath    string // empty unless -reload
	reloadWait    int
	lastReloadErr string
}

// begin starts a new run where start says, with tuning t. Recording then
// holds only this run.
func (g *game) begin(start replay.Replay, t tuning.Tuning) error {
	w, m, err := replay.Start(start, t, g.chunks)
	if err != nil {
		return err
	}
	g.world, g.start, g.startMap, g.recorded = w, start, m, nil
	return nil
}

// restart begins the next run after the last one ended: a new tower unless
// the seed is fixed, the same chunk in chunk mode, and the current tuning,
// re-read first under -reload so an edit saved just before is not missed.
// If the next run cannot start, such as after a reload dropped an enemy the
// chunks use, it says why and stays on the ended run.
func (g *game) restart() {
	if g.reloadPath != "" {
		g.reloadWait = 0
		g.pollTuning()
	}
	next := replay.Replay{Tower: g.start.Tower, Seed: g.start.Seed, Chunk: g.start.Chunk}
	if g.newSeeds {
		next.Seed = uint64(time.Now().UnixNano())
	}
	if err := g.begin(next, g.world.Tuning()); err != nil {
		log.Printf("cannot start the next run: %v", err)
		return
	}
	if g.newSeeds {
		log.Printf("climbing tower %d (replay it with -seed %d)", next.Seed, next.Seed)
	}
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
	if g.world.Over && !g.replaying && input.Restart() {
		g.restart()
		return nil
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
	t := g.world.Tuning()
	msg := fmt.Sprintf("tick %d  fps %.0f\nx %d y %d\nvx %.0f vy %.0f\nfuel %d/%d  hp %d/%d\ncam %.0f",
		g.world.Tick, ebiten.ActualFPS(), p.Body.X, p.Body.Y, p.VX, p.VY,
		p.Fuel, t.Gun.Magazine, p.HP, t.Player.MaxHP, g.world.Camera.Y)
	if g.world.Over {
		msg += "\n\nRUN OVER - press R"
	}
	ebitenutil.DebugPrint(screen, msg)
}

func (g *game) Layout(int, int) (int, int) {
	return ScreenWidth, ScreenHeight
}

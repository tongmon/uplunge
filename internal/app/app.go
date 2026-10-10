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
	// Lab, if set, climbs a lab shaft for trying out the enemy spacing
	// (tuning "lab") instead of a tower. Seed places its enemies.
	Lab bool
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
	g.newSeeds = (start.Tower || start.Lab) && cfg.Seed == nil && cfg.ReplayPath == ""
	if cfg.ReplayPath != "" {
		g.playback = start.Inputs
		g.replaying = true
	} else if g.newSeeds {
		logClimb(start)
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
		out := replay.Replay{Tower: g.start.Tower, Lab: g.start.Lab, Seed: g.start.Seed, Chunk: g.start.Chunk,
			Tuning: tun.Fingerprint(), Map: g.startMap.Fingerprint(), Inputs: g.recorded}
		if err := replay.Save(cfg.RecordPath, out); err != nil {
			return errors.Join(runErr, err)
		}
	}
	return runErr
}

// logClimb says which tower or lab a run climbs and how to climb it again.
func logClimb(r replay.Replay) {
	again := fmt.Sprintf("-seed %d", r.Seed)
	if r.Lab {
		again = "-lab " + again
	}
	log.Printf("climbing %s (climb it again with %s)", r.Where(), again)
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
		case cfg.Seed != nil && (!(r.Tower || r.Lab) || *cfg.Seed != r.Seed):
			return r, fmt.Errorf("app: -seed %d does not match the replay, which starts in %s", *cfg.Seed, r.Where())
		case cfg.Lab && !r.Lab:
			return r, fmt.Errorf("app: -lab does not match the replay, which starts in %s", r.Where())
		}
		return r, nil
	}
	if cfg.Chunk != "" {
		if cfg.Seed != nil || cfg.Lab {
			return replay.Replay{}, fmt.Errorf("app: -chunk plays a single chunk; it takes no -seed or -lab")
		}
		return replay.Replay{Chunk: cfg.Chunk}, nil
	}
	seed := uint64(time.Now().UnixNano())
	if cfg.Seed != nil {
		seed = *cfg.Seed
	}
	return replay.Replay{Tower: !cfg.Lab, Lab: cfg.Lab, Seed: seed}, nil
}

type game struct {
	world *sim.World
	fx    render.Effects
	// start and startMap are where the current run started.
	start    replay.Replay
	startMap *level.TileMap
	chunks   []level.Chunk
	// newSeeds makes each restart climb a new tower from the clock, unless
	// -seed fixed it.
	newSeeds bool
	// The core tuning ratio for the HUD (docs/design.md section 4) is a
	// reach over gap. The reaches are the current tuning's, from the ground
	// and from a stomp (lower bounds unless reachDone); gap is the average
	// between the run's stompers stompable enemies.
	groundReach, stompReach int
	reachDone               bool
	gap                     float64
	stompers                int

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
	g.fx = render.NewEffects()
	g.measureReach(t)
	g.gap, g.stompers = sim.EnemyGap(t, m)
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
	next := replay.Replay{Tower: g.start.Tower, Lab: g.start.Lab, Seed: g.start.Seed, Chunk: g.start.Chunk}
	if g.newSeeds {
		next.Seed = uint64(time.Now().UnixNano())
	}
	if err := g.begin(next, g.world.Tuning()); err != nil {
		log.Printf("cannot start the next run: %v", err)
		return
	}
	if g.newSeeds {
		logClimb(next)
	}
}

// measureReach updates the HUD's reaches for tuning t.
func (g *game) measureReach(t tuning.Tuning) {
	ground, gDone := sim.GroundReach(t)
	stomp, sDone, _ := sim.StompReach(t)
	g.groundReach, g.stompReach, g.reachDone = ground, stomp, gDone && sDone
}

// canRestart reports whether R starts the next run now: once a run is over
// or cleared, and at any time in a lab run, which is for trying out
// numbers. A replay never restarts.
func (g *game) canRestart() bool {
	return !g.replaying && (g.world.Over || g.world.Cleared || g.start.Lab)
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
	if g.canRestart() && input.Restart() {
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
	g.fx.Step(g.world)
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
	if applied {
		// Part of the tuning may apply even when err says what was kept.
		g.measureReach(g.world.Tuning())
	}
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
	render.World(screen, g.world, &g.fx)
	p := g.world.Player
	t := g.world.Tuning()
	msg := fmt.Sprintf("tick %d  fps %.0f\nx %d y %d\nvx %.0f vy %.0f\nfuel %d/%d  hp %d/%d\ncam %.0f",
		g.world.Tick, ebiten.ActualFPS(), p.Body.X, p.Body.Y, p.VX, p.VY,
		p.Fuel, t.Gun.Magazine, p.HP, t.Player.MaxHP, g.world.Camera.Y)
	if wt := g.world.Water; wt.On {
		msg += fmt.Sprintf("  water %+.0f", wt.Y-float64(p.Body.Y+p.Body.H))
	}
	reach := fmt.Sprintf("%d/%d", g.groundReach, g.stompReach)
	if !g.reachDone {
		reach = ">=" + reach // a measurement hit its step limit
	}
	switch {
	case g.stompers < 2:
		msg += fmt.Sprintf("\nratio - (reach %s, %d stompable enemies)", reach, g.stompers)
	case g.gap == 0:
		msg += fmt.Sprintf("\nratio - (reach %s, enemies all at one height)", reach)
	default:
		msg += fmt.Sprintf("\nratio %.2f floor %.2f stomp\n  = reach %s / gap %.0f",
			float64(g.groundReach)/g.gap, float64(g.stompReach)/g.gap, reach, g.gap)
	}
	switch {
	case g.world.Over:
		msg += "\n\nRUN OVER - press R"
	case g.world.Cleared:
		msg += fmt.Sprintf("\n\nCLEAR in %.1f s with %d HP - press R", float64(g.world.ClearTick)/sim.Hz, p.HP)
	}
	ebitenutil.DebugPrint(screen, msg)
}

func (g *game) Layout(int, int) (int, int) {
	return ScreenWidth, ScreenHeight
}

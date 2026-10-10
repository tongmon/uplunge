// Package sim is the deterministic game simulation. It advances in fixed
// 60 Hz steps and must not depend on Ebitengine, wall-clock time, or any other
// source of nondeterminism.
package sim

import (
	"fmt"

	"github.com/tongmon/uplunge/internal/collide"
	"github.com/tongmon/uplunge/internal/level"
	"github.com/tongmon/uplunge/internal/tuning"
)

// Hz is the fixed simulation rate in steps per second.
const Hz = 60

// Dt is the duration of one simulation step in seconds.
const Dt = 1.0 / Hz

// Input is the player input sampled for a single step.
type Input struct {
	Left   bool
	Right  bool
	Button bool
}

// World is the complete simulation state.
type World struct {
	// Tick counts the steps taken since the world was created.
	Tick   uint64
	Map    *level.TileMap
	Player Player

	// tuning is read every step; change it with SetTuning.
	tuning tuning.Tuning
}

// NewWorld returns a world at tick zero with the player's top-left corner at
// (x, y).
func NewWorld(t tuning.Tuning, m *level.TileMap, x, y int) *World {
	return &World{
		tuning: t,
		Map:    m,
		Player: newPlayer(t.Player, x, y),
	}
}

// Step advances the world by exactly one fixed timestep of Dt seconds.
func (w *World) Step(in Input) {
	w.Player.step(in, w.tuning.Player, w.Map)
	w.Tick++
}

// NewWorldInChunk returns a world with the player dropped in at the top
// centre of m. Chunks carry no start position, so the spot must be open for
// the whole hitbox.
func NewWorldInChunk(t tuning.Tuning, m *level.TileMap) (*World, error) {
	p := t.Player
	x := (m.Cols*m.TileSize - p.Width) / 2
	if p.Height > m.Rows*m.TileSize || collide.Overlaps(m, x, 0, p.Width, p.Height) {
		return nil, fmt.Errorf("sim: no room to spawn a %dx%d player at the top centre (%d, 0)",
			p.Width, p.Height, x)
	}
	return NewWorld(t, m, x, 0), nil
}

// Tuning returns the tuning the world currently steps with.
func (w *World) Tuning() tuning.Tuning {
	return w.tuning
}

// SetTuning replaces the tuning from the next step on. A new player size is
// applied around the middle of the player's feet. If the resized hitbox would
// overlap a solid, the old size is kept and an error says so; every other
// value is still applied, and calling SetTuning again later retries the size.
func (w *World) SetTuning(t tuning.Tuning) error {
	var err error
	b := &w.Player.Body
	if p := t.Player; p.Width != b.W || p.Height != b.H {
		if !b.Resize(w.Map, p.Width, p.Height) {
			err = fmt.Errorf("sim: kept the %dx%d player size: %dx%d would overlap a solid here",
				b.W, b.H, p.Width, p.Height)
			t.Player.Width, t.Player.Height = b.W, b.H
		}
	}
	w.tuning = t
	return err
}

// Package sim is the deterministic game simulation. It advances in fixed
// 60 Hz steps and must not depend on Ebitengine, wall-clock time, or any other
// source of nondeterminism.
package sim

import (
	"errors"
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
	// Bullets are the player's shots in flight, oldest first.
	Bullets []Bullet
	Camera  Camera

	// tuning is read every step; change it with SetTuning.
	tuning tuning.Tuning
	// blocks is tuning.Blocks by tile value.
	blocks *blockTable
}

// NewWorld returns a world at tick zero with the player's top-left corner at
// (x, y). The world plays on its own copy of m, so breaking blocks leaves m
// unchanged. Every tile value in m must have a block definition in t.
func NewWorld(t tuning.Tuning, m *level.TileMap, x, y int) *World {
	t = t.Clone()
	w := &World{
		tuning: t,
		Map:    m.Clone(),
		Player: newPlayer(t, x, y),
		blocks: newBlockTable(t.Blocks),
	}
	w.blocks.apply(w.Map)
	w.Camera.Y = w.cameraTarget()
	return w
}

// Step advances the world by exactly one fixed timestep of Dt seconds.
func (w *World) Step(in Input) {
	w.stepBullets()
	if w.Player.step(in, w.tuning, w.Map, w.blocks) {
		w.spawnBullet()
	}
	w.stepCamera()
	w.Tick++
}

// NewWorldInChunk returns a world with the player dropped in at the top
// centre of m. Chunks carry no start position, so the spot must be open for
// the whole hitbox.
func NewWorldInChunk(t tuning.Tuning, m *level.TileMap) (*World, error) {
	if err := newBlockTable(t.Blocks).check(m); err != nil {
		return nil, err
	}
	p := t.Player
	x := (m.Cols*m.TileSize - p.Width) / 2
	w := NewWorld(t, m, x, 0)
	// Check on the world's map, which has the block shapes applied.
	if p.Height > m.Rows*m.TileSize || collide.Overlaps(w.Map, x, 0, p.Width, p.Height) {
		return nil, fmt.Errorf("sim: no room to spawn a %dx%d player at the top centre (%d, 0)",
			p.Width, p.Height, x)
	}
	return w, nil
}

// NewWorldInTower returns a world with the player standing on the floor at
// the bottom centre of m, a tower built with level.BuildTower.
func NewWorldInTower(t tuning.Tuning, m *level.TileMap) (*World, error) {
	if err := newBlockTable(t.Blocks).check(m); err != nil {
		return nil, err
	}
	p := t.Player
	x := (m.Cols*m.TileSize - p.Width) / 2
	y := (m.Rows-1)*m.TileSize - p.Height
	w := NewWorld(t, m, x, y)
	if b := w.Player.Body; y < 0 || collide.Overlaps(w.Map, x, y, p.Width, p.Height) || !b.OnGround(w.Map) {
		return nil, fmt.Errorf("sim: no room to stand a %dx%d player on the bottom centre (%d, %d)",
			p.Width, p.Height, x, y)
	}
	return w, nil
}

// Tuning returns a copy of the tuning the world currently steps with.
func (w *World) Tuning() tuning.Tuning {
	return w.tuning.Clone()
}

// SetTuning replaces the tuning from the next step on. A new player size is
// applied around the middle of the player's feet. If the resized hitbox would
// overlap a solid, the old size is kept and an error says so; likewise the old
// block definitions are kept if the new ones leave a tile value of the map
// undefined or would turn a tile the player is inside solid. Every other value
// is still applied, and calling SetTuning again later retries what was kept.
func (w *World) SetTuning(t tuning.Tuning) error {
	var errs []error
	t = t.Clone()
	b := &w.Player.Body
	if err := w.setBlocks(t.Blocks); err != nil {
		errs = append(errs, fmt.Errorf("sim: kept the block definitions: %w", err))
		t.Blocks = w.tuning.Blocks
	}
	if p := t.Player; p.Width != b.W || p.Height != b.H {
		if !b.Resize(w.Map, p.Width, p.Height) {
			errs = append(errs, fmt.Errorf("sim: kept the %dx%d player size: %dx%d would overlap a solid here",
				b.W, b.H, p.Width, p.Height))
			t.Player.Width, t.Player.Height = b.W, b.H
		}
	}
	w.Player.Fuel = min(w.Player.Fuel, t.Gun.Magazine)
	w.tuning = t
	return errors.Join(errs...)
}

// setBlocks applies new block definitions to the map, or leaves the old ones
// in place and says why.
func (w *World) setBlocks(blocks []tuning.Block) error {
	bt := newBlockTable(blocks)
	if err := bt.check(w.Map); err != nil {
		return err
	}
	bt.apply(w.Map)
	if b := w.Player.Body; collide.Overlaps(w.Map, b.X, b.Y, b.W, b.H) {
		w.blocks.apply(w.Map)
		return fmt.Errorf("a tile the player is inside would become solid")
	}
	w.blocks = bt
	return nil
}

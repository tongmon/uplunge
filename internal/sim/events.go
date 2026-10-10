package sim

import "github.com/tongmon/uplunge/internal/level"

// MaxBroken caps the broken cells one step reports.
const MaxBroken = 8

// BrokenCell is a tile broken during a step: where it was, what it was, and
// whether a bullet broke it (from above) rather than the head (from below).
type BrokenCell struct {
	Col, Row int
	Tile     level.Tile
	ByBullet bool
}

// Events reports what happened during the last step, for effects that live
// outside the simulation, such as screen shake and sound.
type Events struct {
	// Frozen steps changed nothing but the tick: a freeze was running.
	Frozen bool

	Jumped bool
	Shot   bool
	// Landed is set on the step the player came down on the ground, with
	// LandSpeed the downward speed it hit the ground with.
	Landed    bool
	LandSpeed float64
	// Refilled is set when the magazine gained fuel, by any cause.
	Refilled bool
	// Stomped, Hit (a bullet hit an enemy), and Drilled each freeze the
	// steps that follow, as the tuning's Feel says.
	Stomped bool
	Hit     bool
	Drilled bool
	// Hurt is set when the player lost HP; Caught when the water launched
	// the player.
	Hurt   bool
	Caught bool

	// Broken holds the first NBroken tiles broken this step, for debris.
	// It is an array, not a slice, so Events stays comparable.
	Broken  [MaxBroken]BrokenCell
	NBroken int
}

// addBroken records a broken tile, dropping it once MaxBroken are recorded.
func (e *Events) addBroken(c BrokenCell) {
	if e.NBroken < MaxBroken {
		e.Broken[e.NBroken] = c
		e.NBroken++
	}
}

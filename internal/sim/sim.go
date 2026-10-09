// Package sim is the deterministic game simulation. It advances in fixed
// 60 Hz steps and must not depend on Ebitengine, wall-clock time, or any other
// source of nondeterminism.
package sim

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
	Tick uint64
}

// NewWorld returns a world at tick zero.
func NewWorld() *World {
	return &World{}
}

// Step advances the world by exactly one fixed timestep of Dt seconds.
func (w *World) Step(in Input) {
	w.Tick++
}

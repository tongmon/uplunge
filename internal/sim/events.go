package sim

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
	// Stomped and Drilled freeze the following FreezeTime of steps.
	Stomped bool
	Drilled bool
	// Hurt is set when the player lost HP; Caught when the water launched
	// the player.
	Hurt   bool
	Caught bool
}

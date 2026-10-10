// Package input maps devices to sim.Input.
package input

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/tongmon/uplunge/internal/sim"
)

// Temporary keyboard layout until the M1 playtest settles the default keys.
var (
	leftKeys   = []ebiten.Key{ebiten.KeyArrowLeft, ebiten.KeyA}
	rightKeys  = []ebiten.Key{ebiten.KeyArrowRight, ebiten.KeyD}
	buttonKeys = []ebiten.Key{ebiten.KeyZ, ebiten.KeySpace}
)

// Read samples the keyboard for the current step.
func Read() sim.Input {
	return sim.Input{
		Left:   anyPressed(leftKeys),
		Right:  anyPressed(rightKeys),
		Button: anyPressed(buttonKeys),
	}
}

func anyPressed(keys []ebiten.Key) bool {
	for _, k := range keys {
		if ebiten.IsKeyPressed(k) {
			return true
		}
	}
	return false
}

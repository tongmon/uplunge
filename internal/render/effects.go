package render

import (
	"math"

	"github.com/tongmon/uplunge/internal/sim"
)

// Effects is what the screen adds on top of the simulation to sell impacts:
// screen shake, the player's squash and stretch, and the fuel lamp's flash.
// It changes only through Step, once per simulation step, so a replay draws
// the same frames every time. It holds no Ebitengine state.
type Effects struct {
	// shakeLeft is the seconds of shake left; shakeClock the seconds since
	// the offset last flipped side, and shakeSide the side it is on (0 before
	// the first flip of a shake).
	shakeLeft  float64
	shakeClock float64
	shakeSide  int
	shakeY     int

	scaleX, scaleY float64

	// flashSteps is the steps of lamp flash left after this one; flashing
	// is whether this step shows it.
	flashSteps int
	flashing   bool
}

// NewEffects returns effects at rest.
func NewEffects() Effects {
	return Effects{scaleX: 1, scaleY: 1}
}

// Step advances the effects by one simulation step of w, reacting to the
// step's events. A frozen step changes nothing, as the freeze stops the
// whole scene.
func (fx *Effects) Step(w *sim.World) {
	ev := w.Events
	if ev.Frozen {
		return
	}
	f := w.Tuning().Feel

	// Shake: firing (down) shakes the view along the shot. The offset flips
	// side every ShakeInterval, ShakeScale px per second left, rounded up.
	if ev.Shot {
		fx.shakeLeft = max(fx.shakeLeft, f.ShakeTime)
		fx.shakeSide = 0
		fx.shakeClock = f.ShakeInterval // flip on this step
	}
	if fx.shakeLeft > 0 {
		if fx.shakeClock >= f.ShakeInterval {
			fx.shakeClock -= f.ShakeInterval
			if fx.shakeSide == 0 {
				fx.shakeSide = 1
			} else {
				fx.shakeSide = -fx.shakeSide
			}
			// The shot goes down, so the view is pushed opposite to it first.
			fx.shakeY = -fx.shakeSide * int(math.Ceil(fx.shakeLeft*f.ShakeScale))
		}
		fx.shakeClock += sim.Dt
		fx.shakeLeft -= sim.Dt
	} else {
		fx.shakeY = 0
	}

	// Squash and stretch ease back to 1; a jump or a landing sets them anew.
	fx.scaleX = approach(fx.scaleX, 1, f.Recover*sim.Dt)
	fx.scaleY = approach(fx.scaleY, 1, f.Recover*sim.Dt)
	if ev.Jumped {
		fx.scaleX, fx.scaleY = f.JumpX, f.JumpY
	}
	if ev.Landed {
		s := min(ev.LandSpeed/f.LandSpeed, 1)
		fx.scaleX = 1 + (f.LandX-1)*s
		fx.scaleY = 1 + (f.LandY-1)*s
	}

	// The lamp flashes white for FlashTime on a refill, counting this step.
	if ev.Refilled {
		fx.flashSteps = max(1, int(math.Round(f.FlashTime*sim.Hz)))
	}
	if fx.flashSteps > 0 {
		fx.flashSteps--
		fx.flashing = true
	} else {
		fx.flashing = false
	}
}

// ShakeY is the vertical screen offset of the shake, in whole pixels.
func (fx *Effects) ShakeY() int {
	return fx.shakeY
}

// Scale is how much to stretch the drawn player, around the middle of its
// feet.
func (fx *Effects) Scale() (x, y float64) {
	return fx.scaleX, fx.scaleY
}

// Flashing reports whether the fuel lamp shows the refill flash.
func (fx *Effects) Flashing() bool {
	return fx.flashing
}

func approach(v, target, step float64) float64 {
	if v < target {
		return min(v+step, target)
	}
	return max(v-step, target)
}

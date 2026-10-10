package render

import (
	"math"

	"github.com/tongmon/uplunge/internal/sim"
	"github.com/tongmon/uplunge/internal/tuning"
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
	// shakeScale is the px per second left of the shake running.
	shakeScale float64

	scaleX, scaleY float64

	// flashSteps is the steps of lamp flash left after this one; flashing
	// is whether this step shows it.
	flashSteps int
	flashing   bool

	// Debris is the pieces of broken blocks in flight, oldest first.
	Debris []Piece
}

// Piece is one bit of a broken block, in world pixels.
type Piece struct {
	X, Y, VX, VY float64
	// Size is the edge of the square piece.
	Size float64
	// Tile is the broken block's value, for its color.
	Tile uint8
	// Steps is how many steps it has left.
	Steps int
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
		fx.startShake(f.ShakeTime, f.ShakeScale, f.ShakeInterval)
	}
	// A stomp shakes the view too, as Downwell does on every stomp, and so
	// does the head breaking a block.
	if ev.Stomped {
		fx.startShake(f.StompShakeTime, f.StompShakeScale, f.ShakeInterval)
	}
	if ev.Drilled {
		fx.startShake(f.DrillShakeTime, f.DrillShakeScale, f.ShakeInterval)
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
			fx.shakeY = -fx.shakeSide * int(math.Ceil(fx.shakeLeft*fx.shakeScale))
		}
		fx.shakeClock += sim.Dt
		fx.shakeLeft -= sim.Dt
	} else {
		fx.shakeY = 0
	}

	// Squash and stretch ease back to 1; a jump or a landing sets them anew.
	fx.scaleX = approach(fx.scaleX, 1, f.Recover*sim.Dt)
	fx.scaleY = approach(fx.scaleY, 1, f.Recover*sim.Dt)
	if ev.Landed {
		s := min(ev.LandSpeed/f.LandSpeed, 1)
		fx.scaleX = 1 + (f.LandX-1)*s
		fx.scaleY = 1 + (f.LandY-1)*s
	}
	// A stomp, a drill break, and the water's launch stretch like a jump,
	// as Celeste's Bounce does, and win over a landing in the same step: the
	// player leaves going up.
	if ev.Jumped || ev.Stomped || ev.Drilled || ev.Caught {
		fx.scaleX, fx.scaleY = f.JumpX, f.JumpY
	}

	fx.stepDebris(f.DebrisGravity)
	for i := 0; i < ev.NBroken; i++ {
		fx.burst(ev.Broken[i], w.Map.TileSize, f)
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

// burst breaks a tile into 2×2 pieces, DebrisGap apart around the tile's
// middle so the break reads even while frozen, thrown away from the hit: up
// for the head breaking it from below, down for a bullet from above, and
// outward from the middle. Speeds depend only on the piece's place, so a
// replay bursts the same.
func (fx *Effects) burst(c sim.BrokenCell, tileSize int, f tuning.Feel) {
	size, gap := f.DebrisSize, f.DebrisGap
	midX := float64(c.Col*tileSize) + float64(tileSize)/2
	midY := float64(c.Row*tileSize) + float64(tileSize)/2
	for i := 0; i < 4; i++ {
		side, low := float64(i%2*2-1), i/2 == 1 // -1 left, 1 right; top or bottom row
		x := midX + side*gap/2 + (side-1)*size/2
		y := midY - gap/2 - size
		if low {
			y = midY + gap/2
		}
		vy := -f.DebrisSpeed // the head breaks from below: pieces fly up
		if c.ByBullet {
			vy = f.DebrisSpeed * f.DebrisBulletMult // a bullet from above knocks them down
		}
		if low != c.ByBullet {
			vy *= f.DebrisFarMult // the row away from the hit flies less
		}
		fx.Debris = append(fx.Debris, Piece{
			X: x, Y: y, VX: side * f.DebrisSpeed / 2, VY: vy, Size: size,
			Tile: uint8(c.Tile), Steps: max(1, int(math.Round(f.DebrisLife*sim.Hz))),
		})
	}
}

// stepDebris moves the debris one step and drops the pieces whose time is
// up.
func (fx *Effects) stepDebris(gravity float64) {
	kept := fx.Debris[:0]
	for _, p := range fx.Debris {
		if p.Steps--; p.Steps <= 0 {
			continue
		}
		p.VY += gravity * sim.Dt
		p.X += p.VX * sim.Dt
		p.Y += p.VY * sim.Dt
		kept = append(kept, p)
	}
	fx.Debris = kept
}

// startShake starts a shake of time seconds at scale px per second left,
// unless the shake running is stronger right now: one shake runs at a time,
// whole, never one's time with another's scale.
func (fx *Effects) startShake(time, scale, interval float64) {
	if fx.shakeLeft > 0 && fx.shakeLeft*fx.shakeScale > time*scale {
		return
	}
	fx.shakeLeft, fx.shakeScale = time, scale
	fx.shakeSide = 0
	fx.shakeClock = interval // flip on this step
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

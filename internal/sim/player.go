package sim

import (
	"math"

	"github.com/tongmon/uplunge/internal/collide"
	"github.com/tongmon/uplunge/internal/level"
	"github.com/tongmon/uplunge/internal/tuning"
)

// Player is the grey-box player: run, fall, and a variable-height jump.
type Player struct {
	Body collide.Body
	// VX and VY are the velocity in px/s. VY is negative going up.
	VX, VY float64
	// OnGround reports whether the player stood on a solid after the last step.
	OnGround bool

	// The timers below count whole steps rather than seconds, so each lasts
	// exactly its tuned time instead of drifting with float error.

	// jumpHoldSteps is the number of steps left during which holding the
	// button keeps VY at -JumpSpeed.
	jumpHoldSteps int
	// coyoteSteps is the number of airborne steps left in which a jump is
	// still allowed after walking off a ledge.
	coyoteSteps int
	// bufferSteps is the number of steps left in which an earlier press still
	// starts a jump once one is allowed.
	bufferSteps int
	prevButton  bool
}

func newPlayer(p tuning.Player, x, y int) Player {
	return Player{Body: collide.Body{X: x, Y: y, W: p.Width, H: p.Height}}
}

func (pl *Player) step(in Input, p tuning.Player, m *level.TileMap) {
	b := &pl.Body
	grounded := collide.Overlaps(m, b.X, b.Y+1, b.W, b.H)
	pressed := in.Button && !pl.prevButton
	pl.prevButton = in.Button

	// Run.
	dir := 0.0
	if in.Left {
		dir--
	}
	if in.Right {
		dir++
	}
	accel := p.RunAccel
	if !grounded {
		accel *= p.AirAccelMult
	}
	pl.VX = approach(pl.VX, dir*p.RunSpeed, accel*Dt)

	// Fall. Gravity is skipped on the ground; otherwise the sub-pixel pull
	// into the floor makes VY flicker between 0 and one step of gravity.
	// Holding the button near the top of an arc softens gravity there.
	if !grounded {
		g := p.Gravity
		if in.Button && math.Abs(pl.VY) < p.ApexGravThreshold {
			g *= p.ApexGravMult
		}
		pl.VY = approach(pl.VY, p.MaxFall, g*Dt)
	}

	// Jump hold: holding the button keeps the launch speed for up to
	// JumpHoldTime, counting the launch step; releasing early ends the hold
	// and gives a lower jump.
	if pl.jumpHoldSteps > 0 {
		if in.Button {
			pl.VY = min(pl.VY, -p.JumpSpeed)
			pl.jumpHoldSteps--
		} else {
			pl.jumpHoldSteps = 0
		}
	}

	// Jump: a press is remembered for JumpBufferTime, and a jump is allowed
	// on the ground and for CoyoteTime after walking off a ledge.
	if grounded {
		pl.coyoteSteps = steps(p.CoyoteTime)
	}
	if pressed {
		pl.bufferSteps = steps(p.JumpBufferTime)
	}
	if pl.bufferSteps > 0 && pl.coyoteSteps > 0 {
		pl.VY = -p.JumpSpeed
		// A buffered tap already released gets no hold, so a fresh press
		// right after the launch cannot stretch it into a full jump.
		pl.jumpHoldSteps = 0
		if in.Button {
			pl.jumpHoldSteps = steps(p.JumpHoldTime) - 1
		}
		pl.bufferSteps = 0
		pl.coyoteSteps = 0
	}
	if pl.bufferSteps > 0 {
		pl.bufferSteps--
	}
	if !grounded && pl.coyoteSteps > 0 {
		pl.coyoteSteps--
	}

	if b.MoveX(m, pl.VX*Dt) {
		pl.VX = 0
	}
	if b.MoveY(m, pl.VY*Dt) {
		pl.VY = 0
		pl.jumpHoldSteps = 0
	}
	pl.OnGround = collide.Overlaps(m, b.X, b.Y+1, b.W, b.H)
}

// steps converts a tuned time to whole steps, at least one.
func steps(seconds float64) int {
	return max(1, int(math.Round(seconds*Hz)))
}

// approach moves v toward target by at most step.
func approach(v, target, step float64) float64 {
	if v < target {
		return min(v+step, target)
	}
	return max(v-step, target)
}

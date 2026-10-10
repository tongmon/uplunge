package sim

import (
	"math"

	"github.com/tongmon/uplunge/internal/collide"
	"github.com/tongmon/uplunge/internal/level"
	"github.com/tongmon/uplunge/internal/tuning"
)

// Player is the grey-box player: run, fall, a variable-height jump, and the
// gunjet that fires down to push the player up.
type Player struct {
	Body collide.Body
	// VX and VY are the velocity in px/s. VY is negative going up.
	VX, VY float64
	// OnGround reports whether the player stood on a solid after the last step.
	OnGround bool
	// Fuel is the number of shots left in the magazine.
	Fuel int

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
	// fireCooldown is the number of steps left until the next shot.
	fireCooldown int
	// firing is set by a press in the air that did not jump and lasts while
	// the button stays down, so the press that starts a jump never fires.
	firing     bool
	prevButton bool
}

func newPlayer(t tuning.Tuning, x, y int) Player {
	return Player{
		Body: collide.Body{X: x, Y: y, W: t.Player.Width, H: t.Player.Height},
		Fuel: t.Gun.Magazine,
	}
}

// step advances the player by one step and reports whether it fired a shot.
func (pl *Player) step(in Input, t tuning.Tuning, m *level.TileMap) (shot bool) {
	p, g := t.Player, t.Gun
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
	jumped := false
	if pl.bufferSteps > 0 && pl.coyoteSteps > 0 {
		jumped = true
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

	// Gunjet: standing on the ground ends the fire stream. A press in the
	// air that did not jump starts the stream, and holding the button fires
	// every FireInterval while fuel lasts. A shot uses up the press, so it
	// cannot also jump on landing; a press that could not fire stays
	// buffered.
	if pl.fireCooldown > 0 {
		pl.fireCooldown--
	}
	switch {
	case grounded:
		pl.firing = false
	case !in.Button:
		pl.firing = false
	case pressed && !jumped:
		pl.firing = true
	}
	if pl.firing && pl.Fuel > 0 && pl.fireCooldown == 0 {
		pl.VY = min(pl.VY, -g.Thrust)
		pl.Fuel--
		pl.fireCooldown = steps(g.FireInterval)
		pl.bufferSteps = 0
		shot = true
	}

	if b.MoveX(m, pl.VX*Dt) {
		pl.VX = 0
	}
	if b.MoveY(m, pl.VY*Dt) {
		pl.VY = 0
		pl.jumpHoldSteps = 0
	}
	pl.OnGround = collide.Overlaps(m, b.X, b.Y+1, b.W, b.H)
	// Landing refills the magazine on the step it happens.
	if pl.OnGround {
		pl.Fuel = g.Magazine
	}
	return shot
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

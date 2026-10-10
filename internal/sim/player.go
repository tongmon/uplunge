package sim

import (
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

	// jumpHold is the time left during which holding the button keeps VY at
	// -JumpSpeed.
	jumpHold   float64
	prevButton bool
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
	if !grounded {
		pl.VY = approach(pl.VY, p.MaxFall, p.Gravity*Dt)
	}

	// Jump: holding the button keeps the launch speed for up to JumpHoldTime;
	// releasing early ends the hold and gives a lower jump.
	if pl.jumpHold > 0 {
		if in.Button {
			pl.VY = min(pl.VY, -p.JumpSpeed)
			pl.jumpHold -= Dt
		} else {
			pl.jumpHold = 0
		}
	}
	if pressed && grounded {
		pl.VY = -p.JumpSpeed
		pl.jumpHold = p.JumpHoldTime
	}

	if b.MoveX(m, pl.VX*Dt) {
		pl.VX = 0
	}
	if b.MoveY(m, pl.VY*Dt) {
		pl.VY = 0
		pl.jumpHold = 0
	}
	pl.OnGround = collide.Overlaps(m, b.X, b.Y+1, b.W, b.H)
}

// approach moves v toward target by at most step.
func approach(v, target, step float64) float64 {
	if v < target {
		return min(v+step, target)
	}
	return max(v-step, target)
}

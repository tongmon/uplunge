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
	// HP is the hits the player can still take; the run ends at 0.
	HP int

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
	// bounceSteps is the number of steps left during which a stomp keeps
	// VY at -StompSpeed, button or not.
	bounceSteps int
	// invulnSteps is the number of steps left during which hits do nothing.
	invulnSteps int
	// fireCooldown is the number of steps left until the next shot.
	fireCooldown int
	// firing is set by a press in the air that did not jump and lasts while
	// the button stays down, so the press that starts a jump never fires.
	firing     bool
	prevButton bool
	// frozenPress is a press made during a freeze and held since.
	frozenPress bool
	// events collects what the player did this step.
	events Events
}

func newPlayer(t tuning.Tuning, x, y int) Player {
	return Player{
		Body: collide.Body{X: x, Y: y, W: t.Player.Width, H: t.Player.Height},
		Fuel: t.Gun.Magazine,
		HP:   t.Player.MaxHP,
	}
}

// refill fills the magazine, reporting it as a refill when fuel was missing.
func (pl *Player) refill(magazine int) {
	if pl.Fuel < magazine {
		pl.events.Refilled = true
	}
	pl.Fuel = magazine
}

// InvulnSteps is the number of steps left during which hits do nothing. It
// does not count down during a freeze.
func (pl *Player) InvulnSteps() int {
	return pl.invulnSteps
}

// noteFrozenInput follows the button while the world is frozen, so a press
// made during the freeze and still held when it ends counts as a press then.
// One let go before the end is lost: the button is up when the world moves.
func (pl *Player) noteFrozenInput(in Input) {
	if in.Button && !pl.prevButton {
		pl.frozenPress = true
	}
	pl.prevButton = in.Button
}

// Invulnerable reports whether hits do nothing right now.
func (pl *Player) Invulnerable() bool {
	return pl.invulnSteps > 0
}

// stomp bounces the player off an enemy it landed on and refills the
// magazine.
func (pl *Player) stomp(t tuning.Tuning) {
	// A stomp comes after the step's move, so the bounce drives the next
	// StompHoldTime of steps (a jump's launch step moves in the same step).
	pl.VY = -t.Player.StompSpeed
	pl.bounceSteps = steps(t.Player.StompHoldTime)
	pl.jumpHoldSteps = 0
	pl.coyoteSteps = 0
	pl.refill(t.Gun.Magazine)
	pl.events.Stomped = true
}

// hurt takes 1 HP, refills the magazine, knocks the player up and away,
// and makes it immune to hits for a while. (dx, dy) points from what hit
// the player to the player: the sideways knockback is KnockbackX scaled by
// its horizontal part, so a hit from the side pushes hardest and one from
// straight above or below leaves VX as it was. The upward knockback is
// always KnockbackY.
func (pl *Player) hurt(t tuning.Tuning, dx, dy float64) {
	p := t.Player
	pl.HP = max(0, pl.HP-1)
	pl.events.Hurt = true
	pl.refill(t.Gun.Magazine)
	if dx != 0 {
		pl.VX = p.KnockbackX * dx / math.Hypot(dx, dy)
	}
	pl.VY = -p.KnockbackY
	pl.jumpHoldSteps = 0
	pl.bounceSteps = 0
	pl.coyoteSteps = 0
	pl.invulnSteps = steps(p.InvulnTime)
}

// step advances the player by one step and reports whether it fired a shot.
// The head breaks drill blocks in m as defined by bt.
func (pl *Player) step(in Input, t tuning.Tuning, m *level.TileMap, bt *blockTable) (shot bool) {
	p, g := t.Player, t.Gun
	b := &pl.Body
	pl.events = Events{}
	wasOnGround := pl.OnGround
	// Only a player that is not rising stands on something, as in Celeste;
	// otherwise rising through a one-way platform would land on its top edge.
	grounded := pl.VY >= 0 && b.OnGround(m)
	pressed := in.Button && (!pl.prevButton || pl.frozenPress)
	pl.frozenPress = false
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

	if pl.invulnSteps > 0 {
		pl.invulnSteps--
	}

	// Stomp bounce: keeps the bounce speed for StompHoldTime.
	if pl.bounceSteps > 0 {
		pl.VY = min(pl.VY, -p.StompSpeed)
		pl.bounceSteps--
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
		pl.events.Jumped = true
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
		pl.events.Shot = true
		pl.VY = min(pl.VY, -g.Thrust)
		pl.Fuel--
		pl.fireCooldown = steps(g.FireInterval)
		pl.bufferSteps = 0
		shot = true
	}

	if b.MoveX(m, pl.VX*Dt) {
		pl.VX = 0
	}
	fallSpeed := pl.VY
	if b.MoveY(m, pl.VY*Dt) && !(pl.VY < 0 && pl.clearCeiling(p, m, bt)) {
		pl.VY = 0
		pl.jumpHoldSteps = 0
		pl.bounceSteps = 0
	}
	// Rising through a one-way platform gets a push, so a jump that only
	// just reaches one still ends on top.
	if !grounded && pl.VY <= 0 && collide.OverlapsShape(m, b.X, b.Y, b.W, b.H, level.ShapeOneWay) &&
		b.MoveY(m, -p.OneWayAssist*Dt) && !pl.clearCeiling(p, m, bt) {
		pl.VY = 0
		pl.jumpHoldSteps = 0
		pl.bounceSteps = 0
	}
	pl.OnGround = pl.VY >= 0 && b.OnGround(m)
	if pl.OnGround && !wasOnGround {
		pl.events.Landed = true
		pl.events.LandSpeed = max(0, fallSpeed)
	}
	// Landing refills the magazine on the step it happens.
	if pl.OnGround {
		pl.refill(g.Magazine)
	}
	return shot
}

// clearCeiling handles the head hitting something while rising and reports
// whether the player may keep rising. The head first breaks the drill blocks
// right above it, which bounces the player up when that clears the way.
// Otherwise a ceiling corner within CornerCorrection pixels is slipped past
// by moving sideways, trying the side the player is moving toward.
func (pl *Player) clearCeiling(p tuning.Player, m *level.TileMap, bt *blockTable) bool {
	b := &pl.Body
	broke := bt.breakIn(m, b.X, b.Y-1, b.W, 1, byDrill, false, &pl.events)
	if broke {
		pl.events.Drilled = true
	}
	if broke && !collide.Overlaps(m, b.X, b.Y-1, b.W, b.H) {
		pl.VY = min(pl.VY, -p.DrillBounce)
		return true
	}
	for _, dir := range []int{-1, 1} {
		if (dir < 0 && pl.VX > 0) || (dir > 0 && pl.VX < 0) {
			continue
		}
		for k := 1; k <= p.CornerCorrection; k++ {
			if !collide.Overlaps(m, b.X+dir*k, b.Y-1, b.W, b.H) {
				b.X += dir * k
				b.Y--
				return true
			}
		}
	}
	return false
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

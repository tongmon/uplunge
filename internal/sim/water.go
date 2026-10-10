package sim

// Water is the rising water that chases the player up the tower, with the
// rubber-band speed of Celeste's rising lava: it rises at its base speed
// around a baseline just above the bottom of the view, faster the further
// it lags below, slower once it is up on screen, and never further than
// MaxLag below the baseline.
type Water struct {
	// On is false in chunk mode, which has no water.
	On bool
	// Y is the surface in world pixels; the water fills everything below.
	Y float64

	// pauseSteps is the number of steps left during which the water does
	// not rise and does not catch the player again, after it did.
	pauseSteps int
	// The water eases from retreatFrom down to retreatTo over retreatTotal
	// steps after catching the player; retreatLeft counts down.
	retreatFrom, retreatTo    float64
	retreatTotal, retreatLeft int
}

// startWater puts the water StartBelow under the map's bottom edge.
func (w *World) startWater() {
	w.Water = Water{On: true, Y: float64(w.Map.Rows*w.Map.TileSize) + w.tuning.Water.StartBelow}
}

// stepWater moves the water for one step.
func (w *World) stepWater() {
	wt, c := &w.Water, w.tuning.Water
	if !wt.On {
		return
	}
	if wt.pauseSteps > 0 {
		wt.pauseSteps--
	}
	if wt.retreatLeft > 0 {
		wt.retreatLeft--
		t := 1 - float64(wt.retreatLeft)/float64(wt.retreatTotal)
		wt.Y = wt.retreatFrom + (wt.retreatTo-wt.retreatFrom)*cubeOut(t)
	}
	baseline := w.Camera.Bottom() - c.Baseline
	wt.Y = min(wt.Y, baseline+c.MaxLag)
	mult := 1.0
	if wt.Y > baseline {
		mult = 1 + (c.MaxMult-1)*min(1, (wt.Y-baseline)/c.MaxLag)
	} else {
		mult = 1 + (c.MinMult-1)*min(1, (baseline-wt.Y)/c.SlowRange)
	}
	if wt.pauseSteps == 0 {
		wt.Y -= c.Speed * mult * Dt
	}
}

// touchWater is the fall safety net: water reaching the player costs 1 HP
// (unless the player is immune), refills the magazine, launches the player
// up at Bounce, eases the water down by Retreat over RetreatTime, and stops
// it rising for PauseTime, counted from the touch.
func (w *World) touchWater() {
	wt, c, pl := &w.Water, w.tuning.Water, &w.Player
	if !wt.On || wt.pauseSteps > 0 || float64(pl.Body.Y+pl.Body.H) <= wt.Y {
		return
	}
	if pl.invulnSteps == 0 {
		pl.HP = max(0, pl.HP-1)
		pl.invulnSteps = steps(w.tuning.Player.InvulnTime)
	}
	pl.Fuel = w.tuning.Gun.Magazine
	pl.VY = -c.Bounce
	pl.jumpHoldSteps = 0
	pl.bounceSteps = 0
	pl.coyoteSteps = 0
	wt.retreatFrom, wt.retreatTo = wt.Y, wt.Y+c.Retreat
	wt.retreatTotal = steps(c.RetreatTime)
	wt.retreatLeft = wt.retreatTotal
	wt.pauseSteps = steps(c.PauseTime)
}

// cubeOut eases t in [0, 1] fast at first and gently at the end.
func cubeOut(t float64) float64 {
	u := 1 - t
	return 1 - u*u*u
}

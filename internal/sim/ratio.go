package sim

import (
	"slices"

	"github.com/tongmon/uplunge/internal/level"
	"github.com/tongmon/uplunge/internal/tuning"
)

// maxRiseSteps bounds MagazineRise for tunings whose rise does not end in
// time.
const maxRiseSteps = 60 * Hz

// MagazineRise is how many pixels one full magazine lifts the player from
// rest in open air with tuning t, firing as fast as it can: the numerator of
// the core tuning ratio (docs/design.md section 4). done is false when the
// magazine was not empty or the player still rising after maxRiseSteps; the
// rise is then a lower bound.
func MagazineRise(t tuning.Tuning) (rise int, done bool) {
	t = t.Clone()
	t.Enemies = nil
	// Bullets do not move the player; make them vanish at once, so a tuning
	// with absurd bullets cannot make the measurement slow.
	t.Gun.BulletSpeed, t.Gun.BulletLife = 1, Dt
	// Outside a map is open air, so a one-tile map is enough.
	w := NewWorld(t, level.NewTileMap(1, 1, 16), 0, 0)
	top := 0
	for i := 0; i < maxRiseSteps; i++ {
		if w.Player.Fuel == 0 && w.Player.VY >= 0 {
			return -top, true
		}
		w.Step(Input{Button: true})
		top = min(top, w.Player.Body.Y)
	}
	return -top, false
}

// reachSetup returns tuning t made fit for measuring: no bullets to slow it
// down, and only still copies of t's first stompable enemy, which it also
// returns (ok false without one).
func reachSetup(t tuning.Tuning) (tuning.Tuning, tuning.Enemy, bool) {
	t = t.Clone()
	t.Gun.BulletSpeed, t.Gun.BulletLife = 1, Dt
	var e tuning.Enemy
	found := false
	for _, d := range t.Enemies {
		if d.Stompable {
			e, found = d, true
			break
		}
	}
	e.Speed = 0
	t.Enemies = []tuning.Enemy{e}
	return t, e, found
}

// Reach measurement limits: the latest moment tried for the switch to the
// gun, and the steps all tries may take together, so a tuning that never
// comes down cannot stall the game that measures it.
const (
	maxReachStart  = 4 * Hz
	maxReachBudget = 200000
)

// reachTry is one try of a reach measurement: it steps w with the button
// held for the takeoff (hold steps; 0 for none), let go for one step, then
// held for the gun from `start` unfrozen steps on, until the magazine is
// empty and the player no longer rises. It returns the highest the feet got
// and how many steps that took, or ok false if it hit the step limit.
func reachTry(w *World, hold, start int) (top, used int, ok bool) {
	top = w.Player.Body.Y + w.Player.Body.H
	moving := 0 // unfrozen steps taken
	for used < maxRiseSteps {
		frozen := w.freezeSteps > 0
		if !frozen && moving > start && w.Player.Fuel == 0 && w.Player.VY >= 0 {
			return top, used, true
		}
		in := Input{Button: moving < hold || moving > start}
		w.Step(in)
		used++
		if !frozen {
			moving++
		}
		top = min(top, w.Player.Body.Y+w.Player.Body.H)
	}
	return top, used, false
}

// bestReach runs tries from fresh worlds for every moment to switch to the
// gun, up to the takeoff's peak (or maxReachStart), and returns the best
// height above base. done is false when a try or the overall budget ran
// out; the result is then a lower bound.
func bestReach(fresh func() (w *World, base int, ok bool), hold int) (reach int, done, ok bool) {
	budget, done := maxReachBudget, true
	if hold >= maxReachStart {
		// A takeoff this long is not measured to its end.
		hold, done = maxReachStart-1, false
	}
	for start := hold; start < maxReachStart; start++ {
		w, base, ok := fresh()
		if !ok {
			return 0, false, false
		}
		top, used, tryOK := reachTry(w, hold, start)
		reach, done = max(reach, base-top), done && tryOK
		if budget -= used; budget <= 0 {
			return reach, false, true
		}
		// Waiting past the takeoff's peak only loses height.
		if w0, _, _ := fresh(); peaked(w0, hold, start) {
			break
		}
	}
	return reach, done, true
}

// peaked reports whether the takeoff has passed its peak after start
// unfrozen steps: holding the button for hold steps, then letting go.
func peaked(w *World, hold, start int) bool {
	for moving, i := 0, 0; moving <= start && i < maxRiseSteps; i++ {
		frozen := w.freezeSteps > 0
		w.Step(Input{Button: moving < hold})
		if !frozen {
			moving++
		}
	}
	return w.Player.VY > 0
}

// GroundReach is how high above the ground the player's feet can get from
// standing: a full jump, then the whole magazine, starting to fire at the
// best moment (at the jump's peak, as a shot only caps the upward speed).
// It is the reach that counts for the first enemy above a floor.
func GroundReach(t tuning.Tuning) (reach int, done bool) {
	t, _, _ = reachSetup(t)
	t.Enemies = nil
	m := level.NewTileMap(3, 1, 16)
	for c := 0; c < 3; c++ {
		m.Set(c, 0, level.Solid)
	}
	fresh := func() (*World, int, bool) {
		return NewWorld(t, m, 18, -t.Player.Height), 0, true
	}
	r, done, _ := bestReach(fresh, steps(t.Player.JumpHoldTime))
	return r, done
}

// StompReach is how high above a stomped enemy's top the player's feet can
// get: the stomp's bounce, then the whole magazine, starting to fire at the
// best moment. It is the reach that counts from one enemy to the next.
// ok is false when t has no stompable enemy, or the stomp could not be set
// up.
func StompReach(t tuning.Tuning) (reach int, done, ok bool) {
	t, e, ok := reachSetup(t)
	if !ok {
		return 0, true, false
	}
	fresh := func() (*World, int, bool) {
		m := level.NewTileMap(1, 1, 16)
		m.Spawns = []level.Spawn{{Name: e.Name, X: 8, Y: 0}}
		w := NewWorld(t, m, 8-t.Player.Width/2, -e.Height/2-t.Player.Height-1)
		enemyTop := w.Enemies[0].Body.Y
		// Start falling, so the stomp comes at once whatever the gravity.
		w.Player.VY = Hz
		for i := 0; i < 2*Hz && !w.Events.Stomped; i++ {
			w.Step(Input{})
		}
		return w, enemyTop, w.Events.Stomped
	}
	return bestReach(fresh, 0)
}

// EnemyGap is the average vertical distance in pixels between consecutive
// stompable enemies placed in m, from bottom to top: the denominator of the
// core tuning ratio. n is how many stompable enemies there are; the gap is
// 0 when n is below 2.
func EnemyGap(t tuning.Tuning, m *level.TileMap) (gap float64, n int) {
	var ys []int
	for _, s := range m.Spawns {
		if def, ok := enemyDef(t, s.Name); ok && def.Stompable {
			ys = append(ys, s.Y)
		}
	}
	if len(ys) < 2 {
		return 0, len(ys)
	}
	slices.Sort(ys)
	return float64(ys[len(ys)-1]-ys[0]) / float64(len(ys)-1), len(ys)
}

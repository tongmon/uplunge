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

// climb steps w with the button held from step `from` on (and up before
// it), until the magazine is empty and the player no longer rises, and
// returns the highest the feet got. done is false if that took longer than
// maxRiseSteps.
func climb(w *World, from int) (top int, done bool) {
	top = w.Player.Body.Y + w.Player.Body.H
	for i := 0; i < maxRiseSteps; i++ {
		if i > from && w.Player.Fuel == 0 && w.Player.VY >= 0 && !w.Events.Frozen {
			return top, true
		}
		w.Step(Input{Button: i >= from})
		top = min(top, w.Player.Body.Y+w.Player.Body.H)
	}
	return top, false
}

// GroundReach is how high above the ground the player's feet can get from
// standing: a jump, then the whole magazine, switching from the jump to the
// gun at the best moment. It is the reach that counts for the first enemy
// above a floor (the core tuning ratio's numerator from the ground).
func GroundReach(t tuning.Tuning) (reach int, done bool) {
	t, _, _ = reachSetup(t)
	t.Enemies = nil
	m := level.NewTileMap(3, 1, 16)
	for c := 0; c < 3; c++ {
		m.Set(c, 0, level.Solid)
	}
	best, allDone := 0, true
	for hold := 1; hold <= steps(t.Player.JumpHoldTime); hold++ {
		w := NewWorld(t, m, 18, -t.Player.Height)
		// Jump and hold for `hold` steps, let go for one, then fire.
		for i := 0; i < hold; i++ {
			w.Step(Input{Button: true})
		}
		top, ok := climb(w, 1)
		best, allDone = max(best, -top), allDone && ok
	}
	return best, allDone
}

// StompReach is how high above a stomped enemy's top the player's feet can
// get: the stomp's bounce, then the whole magazine, starting to fire at the
// best moment. It is the reach that counts from one enemy to the next.
// ok is false when t has no stompable enemy.
func StompReach(t tuning.Tuning) (reach int, done, ok bool) {
	t, e, ok := reachSetup(t)
	if !ok {
		return 0, true, false
	}
	best, allDone := 0, true
	for wait := 0; wait <= steps(t.Player.StompHoldTime)+steps(t.Gun.FireInterval); wait++ {
		m := level.NewTileMap(1, 1, 16)
		m.Spawns = []level.Spawn{{Name: e.Name, X: 8, Y: 0}}
		w := NewWorld(t, m, 8-t.Player.Width/2, -e.Height/2-t.Player.Height-8)
		enemyTop := w.Enemies[0].Body.Y
		for i := 0; i < Hz && !w.Events.Stomped; i++ {
			w.Step(Input{})
		}
		if !w.Events.Stomped {
			return 0, false, true
		}
		top, ok := climb(w, wait)
		best, allDone = max(best, enemyTop-top), allDone && ok
	}
	return best, allDone, true
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

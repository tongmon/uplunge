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

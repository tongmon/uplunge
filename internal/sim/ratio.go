package sim

import (
	"slices"

	"github.com/tongmon/uplunge/internal/level"
	"github.com/tongmon/uplunge/internal/tuning"
)

// maxRiseSteps bounds MagazineRise for tunings whose rise never ends.
const maxRiseSteps = 60 * Hz

// MagazineRise is how many pixels one full magazine lifts the player from
// rest in open air with tuning t, firing as fast as it can: the numerator of
// the core tuning ratio (docs/design.md section 4).
func MagazineRise(t tuning.Tuning) int {
	t.Enemies = nil
	// Outside a map is open air, so a one-tile map is enough.
	w := NewWorld(t, level.NewTileMap(1, 1, 16), 0, 0)
	top := 0
	for i := 0; i < maxRiseSteps && (w.Player.Fuel > 0 || w.Player.VY < 0); i++ {
		w.Step(Input{Button: true})
		top = min(top, w.Player.Body.Y)
	}
	return -top
}

// EnemyGap is the average vertical distance in pixels between consecutive
// stompable enemies placed in m, from bottom to top: the denominator of the
// core tuning ratio. It is 0 with fewer than two of them.
func EnemyGap(t tuning.Tuning, m *level.TileMap) float64 {
	var ys []int
	for _, s := range m.Spawns {
		if def, ok := enemyDef(t, s.Name); ok && def.Stompable {
			ys = append(ys, s.Y)
		}
	}
	if len(ys) < 2 {
		return 0
	}
	slices.Sort(ys)
	return float64(ys[len(ys)-1]-ys[0]) / float64(len(ys)-1)
}

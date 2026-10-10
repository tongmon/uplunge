package sim

import (
	"fmt"

	"github.com/tongmon/uplunge/internal/collide"
	"github.com/tongmon/uplunge/internal/level"
	"github.com/tongmon/uplunge/internal/tuning"
)

// Enemy is one live enemy.
type Enemy struct {
	// Def is the enemy's definition as it was when the run started; edits to
	// the tuning apply from the next run.
	Def  tuning.Enemy
	Body collide.Body
	// VX is the sideways speed in px/s; it flips at walls.
	VX float64
	// HP is the bullet hits left.
	HP int
}

// enemyDef returns the definition called name.
func enemyDef(t tuning.Tuning, name string) (tuning.Enemy, bool) {
	for _, e := range t.Enemies {
		if e.Name == name {
			return e, true
		}
	}
	return tuning.Enemy{}, false
}

// checkSpawns reports the first spawn in m that names no enemy definition.
func checkSpawns(t tuning.Tuning, m *level.TileMap) error {
	for _, s := range m.Spawns {
		if _, ok := enemyDef(t, s.Name); !ok {
			return fmt.Errorf("sim: spawn %q at (%d, %d) has no enemy definition", s.Name, s.X, s.Y)
		}
	}
	return nil
}

// spawnEnemies places an enemy, centred, at every spawn of m that names an
// enemy definition. Moving enemies start flying right.
func (w *World) spawnEnemies(m *level.TileMap) {
	for _, s := range m.Spawns {
		def, ok := enemyDef(w.tuning, s.Name)
		if !ok {
			continue
		}
		w.Enemies = append(w.Enemies, Enemy{
			Def:  def,
			Body: collide.Body{X: s.X - def.Width/2, Y: s.Y - def.Height/2, W: def.Width, H: def.Height},
			VX:   def.Speed,
			HP:   def.HP,
		})
	}
}

// stepEnemies flies every moving enemy sideways, turning it at walls.
func (w *World) stepEnemies() {
	for i := range w.Enemies {
		e := &w.Enemies[i]
		if e.VX != 0 && e.Body.MoveX(w.Map, e.VX*Dt) {
			e.VX = -e.VX
		}
	}
}

// shootEnemy damages the first enemy the box at (x, y) of size w×h overlaps
// and reports whether there was one. An enemy with no HP left dies.
func (w *World) shootEnemy(x, y, bw, bh int) bool {
	for i := range w.Enemies {
		e := &w.Enemies[i]
		if overlaps(e.Body, collide.Body{X: x, Y: y, W: bw, H: bh}) {
			if e.HP--; e.HP <= 0 {
				w.Enemies = append(w.Enemies[:i], w.Enemies[i+1:]...)
			}
			return true
		}
	}
	return false
}

// touchEnemies resolves the player touching enemies after everyone moved.
// Landing on a stompable enemy from above (feet at or above its top before
// this step) stomps it; any other touch hurts, unless the player is immune.
// At most one enemy is stomped or hurts per step.
func (w *World) touchEnemies(prevBottom int) {
	pl := &w.Player
	for i := range w.Enemies {
		e := &w.Enemies[i]
		if !overlaps(pl.Body, e.Body) {
			continue
		}
		if e.Def.Stompable && prevBottom <= e.Body.Y {
			w.Enemies = append(w.Enemies[:i], w.Enemies[i+1:]...)
			pl.stomp(w.tuning)
			return
		}
		if pl.invulnSteps == 0 {
			dir := 1
			if pl.Body.X*2+pl.Body.W < e.Body.X*2+e.Body.W {
				dir = -1
			}
			pl.hurt(w.tuning, dir)
			return
		}
	}
}

// overlaps reports whether two boxes share any pixel.
func overlaps(a, b collide.Body) bool {
	return a.X < b.X+b.W && b.X < a.X+a.W && a.Y < b.Y+b.H && b.Y < a.Y+a.H
}

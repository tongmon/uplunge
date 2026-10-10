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

// shootEnemy damages the topmost enemy the box overlaps, the first one a
// bullet falling through the box meets, and reports whether there was one.
// An enemy with no HP left dies.
func (w *World) shootEnemy(box collide.Body) bool {
	hit := -1
	for i, e := range w.Enemies {
		if overlaps(e.Body, box) && (hit < 0 || e.Body.Y < w.Enemies[hit].Body.Y) {
			hit = i
		}
	}
	if hit < 0 {
		return false
	}
	if w.Enemies[hit].HP--; w.Enemies[hit].HP <= 0 {
		w.Enemies = append(w.Enemies[:hit], w.Enemies[hit+1:]...)
	}
	return true
}

// enemyInWall reports the first enemy that overlaps a solid tile.
func (w *World) enemyInWall() error {
	for _, e := range w.Enemies {
		if b := e.Body; collide.Overlaps(w.Map, b.X, b.Y, b.W, b.H) {
			return fmt.Errorf("sim: %s at (%d, %d) overlaps a solid tile", e.Def.Name, b.X, b.Y)
		}
	}
	return nil
}

// touchEnemies resolves the player touching enemies after everyone moved,
// the same whatever order the enemies are in. Landing on stompable enemies
// from above (feet at or above their top before this step) stomps all of
// them, and a step with a stomp hurts no one: the safe top stays safe even
// when the player also clips a dangerous enemy. Otherwise touching an enemy
// hurts, unless the player is immune, and knocks the player away from the
// nearest one touched (nearest sideways, between centres).
func (w *World) touchEnemies(prevBottom int) {
	pl := &w.Player
	kept, stomped := w.Enemies[:0], false
	for _, e := range w.Enemies {
		if overlaps(pl.Body, e.Body) && e.Def.Stompable && prevBottom <= e.Body.Y {
			stomped = true
			continue
		}
		kept = append(kept, e)
	}
	w.Enemies = kept
	if stomped {
		pl.stomp(w.tuning)
		return
	}
	if pl.invulnSteps > 0 {
		return
	}
	pc := pl.Body.X*2 + pl.Body.W // twice the centre, to stay in integers
	nearest, dist := -1, 0
	for i, e := range w.Enemies {
		if !overlaps(pl.Body, e.Body) {
			continue
		}
		if d := abs(pc - (e.Body.X*2 + e.Body.W)); nearest < 0 || d < dist {
			nearest, dist = i, d
		}
	}
	if nearest < 0 {
		return
	}
	e, b := w.Enemies[nearest].Body, pl.Body
	// Centre to centre, doubled to stay in integers; only the direction
	// counts.
	pl.hurt(w.tuning, float64(b.X*2+b.W-e.X*2-e.W), float64(b.Y*2+b.H-e.Y*2-e.H))
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// overlaps reports whether two boxes share any pixel.
func overlaps(a, b collide.Body) bool {
	return a.X < b.X+b.W && b.X < a.X+a.W && a.Y < b.Y+b.H && b.Y < a.Y+a.H
}

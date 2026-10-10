package sim

import "github.com/tongmon/uplunge/internal/collide"

// Bullet is a player shot flying straight down.
type Bullet struct {
	Body collide.Body
	// life is the number of moves left before the bullet runs out of range.
	life int
}

// spawnBullet fires a bullet from the middle of the player's feet. A bullet
// that would start inside a solid hits it at once and is not added.
func (w *World) spawnBullet() {
	g, p := w.tuning.Gun, w.Player.Body
	b := collide.Body{
		X: p.X + (p.W-g.BulletWidth)/2,
		Y: p.Y + p.H,
		W: g.BulletWidth,
		H: g.BulletHeight,
	}
	if collide.Overlaps(w.Map, b.X, b.Y, b.W, b.H) {
		return
	}
	w.Bullets = append(w.Bullets, Bullet{Body: b, life: steps(g.BulletLife)})
}

// stepBullets moves every bullet and drops the ones that hit a solid. A
// bullet that used up its last move the step before is dropped first, so it
// covers its full range and is still seen at the end of it.
func (w *World) stepBullets() {
	speed := w.tuning.Gun.BulletSpeed
	kept := w.Bullets[:0]
	for _, b := range w.Bullets {
		if b.life <= 0 || b.Body.MoveY(w.Map, speed*Dt) {
			continue
		}
		b.life--
		kept = append(kept, b)
	}
	w.Bullets = kept
}

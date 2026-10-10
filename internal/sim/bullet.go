package sim

import "github.com/tongmon/uplunge/internal/collide"

// Bullet is a player shot flying straight down.
type Bullet struct {
	Body collide.Body
	// life is the number of moves left before the bullet runs out of range.
	life int
}

// spawnBullet fires a bullet from the middle of the player's feet. A bullet
// that would start inside a solid hits it at once: it breaks what bullets
// break there and is not added.
func (w *World) spawnBullet() {
	g, p := w.tuning.Gun, w.Player.Body
	b := collide.Body{
		X: p.X + (p.W-g.BulletWidth)/2,
		Y: p.Y + p.H,
		W: g.BulletWidth,
		H: g.BulletHeight,
	}
	if collide.Overlaps(w.Map, b.X, b.Y, b.W, b.H) {
		w.blocks.breakIn(w.Map, b.X, b.Y, b.W, b.H, byBullet)
		return
	}
	if w.shootEnemy(b) {
		return
	}
	w.Bullets = append(w.Bullets, Bullet{Body: b, life: steps(g.BulletLife)})
}

// stepBullets moves every bullet and drops the ones that hit a solid,
// breaking the blocks right under them that bullets break, or an enemy,
// damaging it. Bullets pass
// through one-way platforms. A bullet that used up its last move the step
// before is dropped first, so it covers its full range and is still seen at
// the end of it.
func (w *World) stepBullets() {
	speed := w.tuning.Gun.BulletSpeed
	kept := w.Bullets[:0]
	for _, b := range w.Bullets {
		if b.life <= 0 {
			continue
		}
		bb := &b.Body
		from := bb.Y
		hitTile := bb.MoveYThrough(w.Map, speed*Dt)
		// An enemy anywhere on the way, which ends at any tile hit, is met
		// before the tile.
		if w.shootEnemy(collide.Body{X: bb.X, Y: from, W: bb.W, H: bb.Y + bb.H - from}) {
			continue
		}
		if hitTile {
			w.blocks.breakIn(w.Map, bb.X, bb.Y+bb.H, bb.W, 1, byBullet)
			continue
		}
		b.life--
		kept = append(kept, b)
	}
	w.Bullets = kept
}

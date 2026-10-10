// Package collide moves axis-aligned boxes through a tile map in whole pixels.
// It must not depend on Ebitengine.
package collide

import (
	"math"

	"github.com/tongmon/uplunge/internal/level"
)

// Body is an axis-aligned box positioned on whole pixels. Fractional movement
// accumulates in a per-axis remainder until it adds up to a whole pixel.
type Body struct {
	// X and Y are the top-left corner in pixels. Y grows downward.
	X, Y int
	// W and H are the size in pixels.
	W, H int

	remX, remY float64
}

// MoveX moves the body horizontally by dx pixels, one pixel at a time, and
// stops in front of the first solid tile. It reports whether a solid was hit;
// a hit also discards the horizontal remainder.
func (b *Body) MoveX(m *level.TileMap, dx float64) bool {
	n := takeWhole(&b.remX, dx)
	step := sign(n)
	for ; n != 0; n -= step {
		if Overlaps(m, b.X+step, b.Y, b.W, b.H) {
			b.remX = 0
			return true
		}
		b.X += step
	}
	return false
}

// MoveY is MoveX for the vertical axis. Moving down, the body also lands on
// the top edge of one-way platforms.
func (b *Body) MoveY(m *level.TileMap, dy float64) bool {
	return b.moveY(m, dy, true)
}

// MoveYThrough is MoveY that passes through one-way platforms.
func (b *Body) MoveYThrough(m *level.TileMap, dy float64) bool {
	return b.moveY(m, dy, false)
}

func (b *Body) moveY(m *level.TileMap, dy float64, oneWay bool) bool {
	n := takeWhole(&b.remY, dy)
	step := sign(n)
	for ; n != 0; n -= step {
		if Overlaps(m, b.X, b.Y+step, b.W, b.H) ||
			(oneWay && step > 0 && onOneWayTop(m, b.X, b.Y, b.W, b.H)) {
			b.remY = 0
			return true
		}
		b.Y += step
	}
	return false
}

// OnGround reports whether the body stands on a solid or on the top edge of a
// one-way platform.
func (b *Body) OnGround(m *level.TileMap) bool {
	return Overlaps(m, b.X, b.Y+1, b.W, b.H) || onOneWayTop(m, b.X, b.Y, b.W, b.H)
}

// onOneWayTop reports whether the bottom edge of the box at (x, y) rests
// exactly on the top edge of a one-way tile, so moving down would cross it.
func onOneWayTop(m *level.TileMap, x, y, w, h int) bool {
	ts := m.TileSize
	bottom := y + h
	if w <= 0 || h <= 0 || floorMod(bottom, ts) != 0 {
		return false
	}
	r := floorDiv(bottom, ts)
	for c := floorDiv(x, ts); c <= floorDiv(x+w-1, ts); c++ {
		if m.ShapeAt(c, r) == level.ShapeOneWay {
			return true
		}
	}
	return false
}

// Resize changes the size to w×h while keeping the middle of the bottom edge
// in place. A half-pixel shift goes into the horizontal remainder, so resizing
// back restores the exact position. If the new box would overlap a solid,
// nothing changes and Resize returns false.
func (b *Body) Resize(m *level.TileMap, w, h int) bool {
	remX := b.remX + float64(b.W-w)/2
	x := b.X + takeWhole(&remX, 0)
	y := b.Y + b.H - h
	if Overlaps(m, x, y, w, h) {
		return false
	}
	b.X, b.Y, b.W, b.H, b.remX = x, y, w, h, remX
	return true
}

// Overlaps reports whether the box at (x, y) of size w×h overlaps any solid
// tile. Boxes that only touch a tile's edge do not overlap it. One-way
// platforms never count.
func Overlaps(m *level.TileMap, x, y, w, h int) bool {
	return OverlapsShape(m, x, y, w, h, level.ShapeSolid)
}

// OverlapsShape is Overlaps for tiles of shape s.
func OverlapsShape(m *level.TileMap, x, y, w, h int, s level.Shape) bool {
	found := false
	EachTile(m, x, y, w, h, func(c, r int) bool {
		found = m.ShapeAt(c, r) == s
		return !found
	})
	return found
}

// EachTile calls f with every grid cell the box at (x, y) of size w×h
// overlaps, row by row from the top left, until f returns false.
func EachTile(m *level.TileMap, x, y, w, h int, f func(col, row int) bool) {
	if w <= 0 || h <= 0 {
		return
	}
	ts := m.TileSize
	c0, c1 := floorDiv(x, ts), floorDiv(x+w-1, ts)
	r0, r1 := floorDiv(y, ts), floorDiv(y+h-1, ts)
	for r := r0; r <= r1; r++ {
		for c := c0; c <= c1; c++ {
			if !f(c, r) {
				return
			}
		}
	}
}

// takeWhole adds d to *rem and removes and returns the nearest whole number of
// pixels, leaving *rem in [-0.5, 0.5]. Halves round to even, so a leftover of
// exactly ±0.5 rounds to zero next time instead of moving back and forth.
func takeWhole(rem *float64, d float64) int {
	*rem += d
	n := math.RoundToEven(*rem)
	*rem -= n
	return int(n)
}

func sign(n int) int {
	switch {
	case n > 0:
		return 1
	case n < 0:
		return -1
	}
	return 0
}

func floorMod(a, b int) int {
	return a - floorDiv(a, b)*b
}

func floorDiv(a, b int) int {
	q := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		q--
	}
	return q
}

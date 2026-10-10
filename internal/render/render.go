// Package render draws the simulation as grey boxes.
package render

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/tongmon/uplunge/internal/level"
	"github.com/tongmon/uplunge/internal/sim"
	"github.com/tongmon/uplunge/internal/tuning"
)

var (
	backgroundColor = color.Gray{Y: 0x30}
	playerColor     = color.Gray{Y: 0xd0}
	bulletColor     = color.Gray{Y: 0xff}
	// dangerColor marks the sides of an enemy that hurt.
	dangerColor = color.RGBA{R: 0xff, G: 0x30, B: 0x30, A: 0xff}
	// The water keeps a color no zone uses (docs/design.md section 8). It is
	// see-through so the player stays visible under the surface.
	waterColor   = color.RGBA{R: 0x10, G: 0x50, B: 0xa0, A: 0xa0}
	surfaceColor = color.RGBA{R: 0x80, G: 0xd0, B: 0xff, A: 0xff}
	// The goal line marks the top of the map, where a run with a goal
	// clears: alternating squares, like a finish line.
	goalColors = [2]color.RGBA{{R: 0xff, G: 0xd0, B: 0x40, A: 0xff}, {R: 0x40, G: 0x30, B: 0x10, A: 0xff}}
)

// goalSquare is the size of one square of the goal line.
const goalSquare = 4

// surfaceThickness is the height of the bright line on the water surface.
const surfaceThickness = 2

// dangerEdge is the thickness of an enemy's marked dangerous sides.
const dangerEdge = 2

// invulnBlinkSteps is how many steps the player is shown, then hidden,
// while hits do nothing.
const invulnBlinkSteps = 4

// oneWayThickness is how many pixels of a one-way tile are drawn, from its
// top edge, so it reads as a thin platform.
const oneWayThickness = 4

// lampHeight is the height of the helmet lamp drawn on top of the player.
const lampHeight = 4

// World draws the part of w the camera sees onto screen, with fx on top.
func World(screen *ebiten.Image, w *sim.World, fx *Effects) {
	screen.Fill(backgroundColor)

	var colors [256]color.RGBA
	for _, b := range w.Tuning().Blocks {
		r, g, bl := b.RGB()
		colors[b.Value] = color.RGBA{R: r, G: g, B: bl, A: 0xff}
	}

	// Draw on whole pixels so tiles do not shimmer as the camera eases. The
	// shake moves the picture, not the simulation's camera.
	camY := int(math.Round(w.Camera.Y))
	oy := float32(-camY + fx.ShakeY())

	m := w.Map
	ts := float32(m.TileSize)
	r0, r1 := visibleRows(camY, fx.ShakeY(), m.TileSize, m.Rows)
	for r := r0; r <= r1; r++ {
		for c := 0; c < m.Cols; c++ {
			x, y, h := float32(c)*ts, float32(r)*ts+oy, ts
			switch m.ShapeAt(c, r) {
			case level.ShapeEmpty:
				continue
			case level.ShapeOneWay:
				h = oneWayThickness
			}
			vector.FillRect(screen, x, y, ts, h, colors[m.At(c, r)], false)
		}
	}

	if w.Goal {
		for i := 0; i*goalSquare < sim.ViewWidth; i++ {
			for row := 0; row < 2; row++ {
				vector.FillRect(screen, float32(i*goalSquare), float32(row*goalSquare)+oy,
					goalSquare, goalSquare, goalColors[(i+row)%2], false)
			}
		}
	}

	for _, e := range w.Enemies {
		drawEnemy(screen, e, oy)
	}

	// Blink by the immunity left, which a freeze holds, so a frozen frame
	// stays as it is.
	if !w.Player.Invulnerable() || (w.Player.InvulnSteps()/invulnBlinkSteps)%2 == 0 {
		drawPlayer(screen, w, fx, oy)
	}

	for _, bl := range w.Bullets {
		b := bl.Body
		vector.FillRect(screen, float32(b.X), float32(b.Y)+oy, float32(b.W), float32(b.H), bulletColor, false)
	}

	for _, p := range fx.Debris {
		vector.FillRect(screen, float32(math.Round(p.X)), float32(math.Round(p.Y))+oy,
			float32(p.Size), float32(p.Size), colors[p.Tile], false)
	}

	if w.Water.On {
		top := float32(w.Water.Surface()) + oy
		if top < sim.ViewHeight {
			vector.FillRect(screen, 0, top, sim.ViewWidth, sim.ViewHeight-top, waterColor, false)
			vector.FillRect(screen, 0, top, sim.ViewWidth, surfaceThickness, surfaceColor, false)
		}
	}
}

// drawEnemy fills e with its color and marks its dangerous sides: every side,
// or every side but the top for an enemy that is safe to stomp.
func drawEnemy(screen *ebiten.Image, e sim.Enemy, oy float32) {
	r, g, b := e.Def.RGB()
	x, y := float32(e.Body.X), float32(e.Body.Y)+oy
	w, h := float32(e.Body.W), float32(e.Body.H)
	vector.FillRect(screen, x, y, w, h, color.RGBA{R: r, G: g, B: b, A: 0xff}, false)
	const d = dangerEdge
	vector.FillRect(screen, x, y, d, h, dangerColor, false)
	vector.FillRect(screen, x+w-d, y, d, h, dangerColor, false)
	vector.FillRect(screen, x, y+h-d, w, d, dangerColor, false)
	if !e.Def.Stompable {
		vector.FillRect(screen, x, y, w, d, dangerColor, false)
	}
}

// drawPlayer draws the player stretched by fx around the middle of its
// feet, with a helmet lamp on top that shows the fuel: FullColor full,
// shading to EmptyColor at none, white while it flashes for a refill.
func drawPlayer(screen *ebiten.Image, w *sim.World, fx *Effects, oy float32) {
	b, p, t := w.Player.Body, w.Player, w.Tuning()
	sx, sy := fx.Scale()
	bw, bh := float32(float64(b.W)*sx), float32(float64(b.H)*sy)
	x := float32(b.X) + float32(b.W)/2 - bw/2
	y := float32(b.Y+b.H) - bh + oy
	vector.FillRect(screen, x, y, bw, bh, playerColor, false)

	lamp := color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	if !fx.Flashing() {
		frac := float64(p.Fuel) / float64(t.Gun.Magazine)
		r0, g0, b0 := tuning.RGB(t.Feel.EmptyColor)
		r1, g1, b1 := tuning.RGB(t.Feel.FullColor)
		lamp = color.RGBA{R: mix(r0, r1, frac), G: mix(g0, g1, frac), B: mix(b0, b1, frac), A: 0xff}
	}
	vector.FillRect(screen, x, y, bw, min(bh, lampHeight*float32(sy)), lamp, false)
}

// mix blends a toward b by t in [0, 1].
func mix(a, b uint8, t float64) uint8 {
	return uint8(math.Round(float64(a) + (float64(b)-float64(a))*t))
}

// visibleRows returns the first and last map rows on screen with the view's
// top at world y camY and the picture moved down by shake.
func visibleRows(camY, shake, tileSize, rows int) (r0, r1 int) {
	top := camY - shake
	return max(0, floorDiv(top, tileSize)), min(rows-1, floorDiv(top+sim.ViewHeight-1, tileSize))
}

func floorDiv(a, b int) int {
	q := a / b
	if a%b != 0 && a < 0 {
		q--
	}
	return q
}

// Package render draws the simulation as grey boxes.
package render

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/tongmon/uplunge/internal/level"
	"github.com/tongmon/uplunge/internal/sim"
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
)

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

// World draws the part of w the camera sees onto screen.
func World(screen *ebiten.Image, w *sim.World) {
	screen.Fill(backgroundColor)

	var colors [256]color.RGBA
	for _, b := range w.Tuning().Blocks {
		r, g, bl := b.RGB()
		colors[b.Value] = color.RGBA{R: r, G: g, B: bl, A: 0xff}
	}

	// Draw on whole pixels so tiles do not shimmer as the camera eases.
	camY := int(math.Round(w.Camera.Y))
	oy := float32(-camY)

	m := w.Map
	ts := float32(m.TileSize)
	r0 := max(0, camY/m.TileSize)
	r1 := min(m.Rows-1, (camY+sim.ViewHeight)/m.TileSize)
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

	for _, e := range w.Enemies {
		drawEnemy(screen, e, oy)
	}

	b := w.Player.Body
	if !w.Player.Invulnerable() || (w.Tick/invulnBlinkSteps)%2 == 0 {
		vector.FillRect(screen, float32(b.X), float32(b.Y)+oy, float32(b.W), float32(b.H), playerColor, false)
	}

	for _, bl := range w.Bullets {
		b := bl.Body
		vector.FillRect(screen, float32(b.X), float32(b.Y)+oy, float32(b.W), float32(b.H), bulletColor, false)
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

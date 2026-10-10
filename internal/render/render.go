// Package render draws the simulation as grey boxes.
package render

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/tongmon/uplunge/internal/level"
	"github.com/tongmon/uplunge/internal/sim"
)

var (
	backgroundColor = color.Gray{Y: 0x30}
	playerColor     = color.Gray{Y: 0xd0}
	bulletColor     = color.Gray{Y: 0xff}
)

// oneWayThickness is how many pixels of a one-way tile are drawn, from its
// top edge, so it reads as a thin platform.
const oneWayThickness = 4

// World draws w onto screen with the map's top-left at the screen's top-left.
func World(screen *ebiten.Image, w *sim.World) {
	screen.Fill(backgroundColor)

	var colors [256]color.RGBA
	for _, b := range w.Tuning().Blocks {
		r, g, bl := b.RGB()
		colors[b.Value] = color.RGBA{R: r, G: g, B: bl, A: 0xff}
	}

	m := w.Map
	ts := float32(m.TileSize)
	for r := 0; r < m.Rows; r++ {
		for c := 0; c < m.Cols; c++ {
			x, y, h := float32(c)*ts, float32(r)*ts, ts
			switch m.ShapeAt(c, r) {
			case level.ShapeEmpty:
				continue
			case level.ShapeOneWay:
				h = oneWayThickness
			}
			vector.FillRect(screen, x, y, ts, h, colors[m.At(c, r)], false)
		}
	}

	b := w.Player.Body
	vector.FillRect(screen, float32(b.X), float32(b.Y), float32(b.W), float32(b.H), playerColor, false)

	for _, bl := range w.Bullets {
		b := bl.Body
		vector.FillRect(screen, float32(b.X), float32(b.Y), float32(b.W), float32(b.H), bulletColor, false)
	}
}

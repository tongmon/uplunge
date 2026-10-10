package sim

import "math"

// The view is the part of the tower on screen, in pixels: 13 tiles of 16 px
// across, and a fixed height so every device sees the same amount of tower.
const (
	ViewWidth  = 208
	ViewHeight = 360
)

// Camera is where the view sits in the tower. It is part of the simulation
// because what the player can see decides what is fair: the rising water
// follows the bottom of the view.
type Camera struct {
	// Y is the top edge of the view in world pixels. It only ever moves up
	// (decreases), and stays inside the map.
	Y float64
}

// Bottom returns the bottom edge of the view in world pixels.
func (c Camera) Bottom() float64 {
	return c.Y + ViewHeight
}

// cameraTarget is where the camera wants to be: the player's centre at
// Anchor of the view height, shifted ahead by Lookahead while rising.
func (w *World) cameraTarget() float64 {
	c, p := w.tuning.Camera, w.Player
	y := float64(p.Body.Y) + float64(p.Body.H)/2 - c.Anchor*ViewHeight
	if p.VY < 0 {
		y += p.VY * c.Lookahead
	}
	return w.clampCamera(y)
}

// clampCamera keeps the view inside the map. A map shorter than the view
// keeps the view at its top.
func (w *World) clampCamera(y float64) float64 {
	bottom := float64(w.Map.Rows*w.Map.TileSize - ViewHeight)
	return max(0, min(y, bottom))
}

// stepCamera eases the camera toward its target, closing all but
// RemainPerSecond of the gap each second, and never moves it down.
func (w *World) stepCamera() {
	target := w.cameraTarget()
	follow := 1 - math.Pow(w.tuning.Camera.RemainPerSecond, Dt)
	w.Camera.Y = min(w.Camera.Y, w.Camera.Y+(target-w.Camera.Y)*follow)
}

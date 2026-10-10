// Package app wires the simulation to Ebitengine: it owns the window, drives
// the fixed-step loop, and draws the world.
package app

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"github.com/tongmon/uplunge/internal/sim"
)

// Logical screen size in pixels: 13 tiles of 16 px across, fixed height so
// every device sees the same amount of the tower.
const (
	ScreenWidth  = 208
	ScreenHeight = 360
)

var backgroundColor = color.Gray{Y: 0x30}

// Config holds startup options.
type Config struct {
	// Scale is the integer window scale relative to the logical screen.
	Scale int
}

// Run opens the window and blocks until the game exits.
func Run(cfg Config) error {
	if cfg.Scale < 1 {
		return fmt.Errorf("app: scale must be at least 1, got %d", cfg.Scale)
	}
	ebiten.SetWindowTitle("uplunge")
	ebiten.SetWindowSize(ScreenWidth*cfg.Scale, ScreenHeight*cfg.Scale)
	ebiten.SetTPS(sim.Hz)
	ebiten.SetScreenFilterEnabled(false)
	return ebiten.RunGame(&game{world: sim.NewWorld()})
}

type game struct {
	world *sim.World
}

// Update runs exactly one simulation step. Ebitengine calls it sim.Hz times
// per second and catches up with extra calls when a frame runs long.
func (g *game) Update() error {
	g.world.Step(sim.Input{})
	return nil
}

func (g *game) Draw(screen *ebiten.Image) {
	screen.Fill(backgroundColor)
	ebitenutil.DebugPrint(screen, fmt.Sprintf("tick %d\nfps %.0f", g.world.Tick, ebiten.ActualFPS()))
}

func (g *game) Layout(int, int) (int, int) {
	return ScreenWidth, ScreenHeight
}

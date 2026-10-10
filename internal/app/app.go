// Package app wires the simulation to Ebitengine: it owns the window, drives
// the fixed-step loop, and draws the world.
package app

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"github.com/tongmon/uplunge/internal/input"
	"github.com/tongmon/uplunge/internal/level"
	"github.com/tongmon/uplunge/internal/render"
	"github.com/tongmon/uplunge/internal/sim"
	"github.com/tongmon/uplunge/internal/tuning"
)

// Logical screen size in pixels: 13 tiles of 16 px across, fixed height so
// every device sees the same amount of the tower.
const (
	ScreenWidth  = 208
	ScreenHeight = 360
)

// Config holds startup options.
type Config struct {
	// Scale is the integer window scale relative to the logical screen.
	Scale int
	// TuningPath is the tuning JSON file to load.
	TuningPath string
}

// testRoom is a hand-written map used until LDtk chunks load.
var testRoom = []string{
	"#...........#",
	"#...........#",
	"#...........#",
	"#...........#",
	"#...........#",
	"#...........#",
	"#....###....#",
	"#...........#",
	"#...........#",
	"#.###.......#",
	"#...........#",
	"#...........#",
	"#.......###.#",
	"#...........#",
	"#...........#",
	"#...###.....#",
	"#...........#",
	"#...........#",
	"#........####",
	"#...........#",
	"#...........#",
	"#############",
}

// Run opens the window and blocks until the game exits.
func Run(cfg Config) error {
	if cfg.Scale < 1 {
		return fmt.Errorf("app: scale must be at least 1, got %d", cfg.Scale)
	}
	tun, err := tuning.Load(cfg.TuningPath)
	if err != nil {
		return err
	}
	m, err := level.ParseRows(16, testRoom...)
	if err != nil {
		return err
	}
	spawnX := 2 * m.TileSize
	spawnY := (m.Rows-1)*m.TileSize - tun.Player.Height

	ebiten.SetWindowTitle("uplunge")
	ebiten.SetWindowSize(ScreenWidth*cfg.Scale, ScreenHeight*cfg.Scale)
	ebiten.SetTPS(sim.Hz)
	ebiten.SetScreenFilterEnabled(false)
	return ebiten.RunGame(&game{world: sim.NewWorld(tun, m, spawnX, spawnY)})
}

type game struct {
	world *sim.World
}

// Update runs exactly one simulation step. Ebitengine calls it sim.Hz times
// per second and catches up with extra calls when a frame runs long.
func (g *game) Update() error {
	g.world.Step(input.Read())
	return nil
}

func (g *game) Draw(screen *ebiten.Image) {
	render.World(screen, g.world)
	p := g.world.Player
	ebitenutil.DebugPrint(screen, fmt.Sprintf("tick %d  fps %.0f\nx %d y %d\nvx %.0f vy %.0f",
		g.world.Tick, ebiten.ActualFPS(), p.Body.X, p.Body.Y, p.VX, p.VY))
}

func (g *game) Layout(int, int) (int, int) {
	return ScreenWidth, ScreenHeight
}

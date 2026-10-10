// Package app wires the simulation to Ebitengine: it owns the window, drives
// the fixed-step loop, and draws the world.
package app

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"github.com/tongmon/uplunge/internal/collide"
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
	// ChunksPath is the LDtk project holding the level chunks.
	ChunksPath string
	// Chunk names the chunk to play in. Empty means the first chunk.
	Chunk string
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
	chunks, err := level.LoadLDtk(cfg.ChunksPath)
	if err != nil {
		return err
	}
	m, err := pickChunk(chunks, cfg.Chunk)
	if err != nil {
		return err
	}
	spawnX, spawnY, err := spawnPoint(m, tun.Player)
	if err != nil {
		return err
	}

	ebiten.SetWindowTitle("uplunge")
	ebiten.SetWindowSize(ScreenWidth*cfg.Scale, ScreenHeight*cfg.Scale)
	ebiten.SetTPS(sim.Hz)
	ebiten.SetScreenFilterEnabled(false)
	return ebiten.RunGame(&game{world: sim.NewWorld(tun, m, spawnX, spawnY)})
}

// spawnPoint drops the player in at the top centre of the chunk. Chunks carry
// no start position, so the spot must be open for the whole hitbox.
func spawnPoint(m *level.TileMap, p tuning.Player) (x, y int, err error) {
	x = (m.Cols*m.TileSize - p.Width) / 2
	if p.Height > m.Rows*m.TileSize || collide.Overlaps(m, x, y, p.Width, p.Height) {
		return 0, 0, fmt.Errorf("app: no room to spawn a %dx%d player at the top centre (%d, %d)",
			p.Width, p.Height, x, y)
	}
	return x, y, nil
}

func pickChunk(chunks []level.Chunk, name string) (*level.TileMap, error) {
	if name == "" {
		return chunks[0].Map, nil
	}
	names := make([]string, len(chunks))
	for i, c := range chunks {
		if c.Name == name {
			return c.Map, nil
		}
		names[i] = c.Name
	}
	return nil, fmt.Errorf("app: no chunk %q; have %s", name, strings.Join(names, ", "))
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

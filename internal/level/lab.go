package level

import (
	"fmt"

	"github.com/tongmon/uplunge/internal/rng"
)

// BuildLab returns a shaft for trying out the spacing of enemies (the
// denominator of the core tuning ratio): ChunkCols tiles across with a wall
// on each side, rows tall, with a floor at the bottom and nothing else. It
// places the named enemy every spacing px from the floor up to two tiles
// below the top, each moved up or down by up to jitter px and placed at a
// random x between the walls.
func BuildLab(rows int, enemy string, spacing, jitter int, r *rng.Rand) (*TileMap, error) {
	switch {
	case rows < 4:
		return nil, fmt.Errorf("level: a lab needs at least 4 rows, got %d", rows)
	case spacing <= 0:
		return nil, fmt.Errorf("level: lab spacing must be positive, got %d", spacing)
	case jitter < 0 || 2*jitter >= spacing:
		return nil, fmt.Errorf("level: lab jitter must be 0 to under half the spacing %d, got %d", spacing, jitter)
	}
	const ts = ChunkTileSize
	m := NewTileMap(ChunkCols, rows, ts)
	for r := 0; r < rows; r++ {
		m.Set(0, r, Solid)
		m.Set(ChunkCols-1, r, Solid)
	}
	for c := 0; c < ChunkCols; c++ {
		m.Set(c, rows-1, Solid)
	}
	// Centres stay half a tile clear of the walls.
	left, right := ts+ts/2, (ChunkCols-1)*ts-ts/2
	floor := (rows - 1) * ts
	for y := floor - spacing; y > 2*ts; y -= spacing {
		dy := 0
		if jitter > 0 {
			dy = r.IntN(2*jitter+1) - jitter
		}
		x := left + r.IntN(right-left+1)
		m.Spawns = append(m.Spawns, Spawn{Name: enemy, X: x, Y: y + dy})
	}
	return m, nil
}

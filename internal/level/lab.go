package level

import (
	"fmt"

	"github.com/tongmon/uplunge/internal/rng"
)

// BuildLab returns a shaft for trying out the spacing of enemies (the
// denominator of the core tuning ratio): ChunkCols tiles across with a wall
// on each side, rows tall, with a floor at the bottom and nothing else. It
// places the named enemy, whose box is ew×eh, every spacing px up from the
// floor, each moved up or down by up to jitter px and placed at a random x.
// Every enemy box stays between the walls, clear of the floor, and at least
// two tiles below the top.
func BuildLab(rows int, enemy string, ew, eh, spacing, jitter int, r *rng.Rand) (*TileMap, error) {
	const ts = ChunkTileSize
	inner := (ChunkCols - 2) * ts
	switch {
	case rows < 4:
		return nil, fmt.Errorf("level: a lab needs at least 4 rows, got %d", rows)
	case ew <= 0 || eh <= 0 || ew > inner:
		return nil, fmt.Errorf("level: a lab enemy must be 1 to %d px wide and at least 1 px tall, got %dx%d", inner, ew, eh)
	case spacing <= 0:
		return nil, fmt.Errorf("level: lab spacing must be positive, got %d", spacing)
	case jitter < 0 || jitter > (spacing-1)/2:
		// Compared without 2*jitter, which could overflow.
		return nil, fmt.Errorf("level: lab jitter must be 0 to under half the spacing %d, got %d", spacing, jitter)
	case spacing-jitter < eh-eh/2:
		return nil, fmt.Errorf("level: lab spacing %d with jitter %d puts the first enemy too close to the floor", spacing, jitter)
	}
	m := NewTileMap(ChunkCols, rows, ts)
	for r := 0; r < rows; r++ {
		m.Set(0, r, Solid)
		m.Set(ChunkCols-1, r, Solid)
	}
	for c := 0; c < ChunkCols; c++ {
		m.Set(c, rows-1, Solid)
	}
	// Centres keep the whole box inside: half the box from each wall, and
	// the box's top, jitter included, at least two tiles below the top.
	left, right := ts+ew/2, ts+inner-(ew-ew/2)
	floor := (rows - 1) * ts
	for y := floor - spacing; y-jitter-eh/2 >= 2*ts; y -= spacing {
		dy := 0
		if jitter > 0 {
			dy = r.IntN(2*jitter+1) - jitter
		}
		x := left + r.IntN(right-left+1)
		m.Spawns = append(m.Spawns, Spawn{Name: enemy, X: x, Y: y + dy})
	}
	return m, nil
}

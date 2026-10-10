package level

import (
	"fmt"
	"slices"

	"github.com/tongmon/uplunge/internal/rng"
)

// Placement is one chunk in a tower.
type Placement struct {
	Name string
	// Flipped chunks are mirrored left to right.
	Flipped bool
	// Row is the tower row of the chunk's top row.
	Row int
}

// BuildTower stacks chunks into one map, from the bottom up: the base chunk
// first, then length-1 chunks drawn from pool with r. Pool chunks may be
// mirrored, and the same chunk never comes twice in a row (docs/design.md
// section 7). Placements lists the chunks from the bottom up.
func BuildTower(chunks []Chunk, base string, pool []string, length int, r *rng.Rand) (*TileMap, []Placement, error) {
	if length < 1 {
		return nil, nil, fmt.Errorf("level: tower length must be at least 1, got %d", length)
	}
	byName := map[string]*TileMap{}
	for _, c := range chunks {
		byName[c.Name] = c.Map
	}
	for _, name := range append([]string{base}, pool...) {
		if _, ok := byName[name]; !ok {
			_, err := FindChunk(chunks, name) // for its error listing the chunks
			return nil, nil, err
		}
	}
	// Without repeats, the second chunk needs a pool chunk other than the
	// base, and every later one needs two distinct pool chunks to alternate.
	distinct := map[string]bool{}
	for _, name := range pool {
		distinct[name] = true
	}
	if length > 1 && !slices.ContainsFunc(pool, func(n string) bool { return n != base }) {
		return nil, nil, fmt.Errorf("level: a tower of %d chunks needs a pool with a chunk other than the base %q", length, base)
	}
	if length > 2 && len(distinct) < 2 {
		return nil, nil, fmt.Errorf("level: a tower of %d chunks needs at least 2 distinct pool chunks to avoid repeats", length)
	}

	parts := []Placement{{Name: base}}
	for len(parts) < length {
		name := pool[r.IntN(len(pool))]
		if name == parts[len(parts)-1].Name {
			continue
		}
		parts = append(parts, Placement{Name: name, Flipped: r.Bool()})
	}

	rows := 0
	for _, p := range parts {
		rows += byName[p.Name].Rows
	}
	first := byName[base]
	tower := NewTileMap(first.Cols, rows, first.TileSize)
	bottom := rows
	for i := range parts {
		m := byName[parts[i].Name]
		if m.Cols != first.Cols || m.TileSize != first.TileSize {
			return nil, nil, fmt.Errorf("level: chunk %q is %d tiles of %d px across, want %d of %d",
				parts[i].Name, m.Cols, m.TileSize, first.Cols, first.TileSize)
		}
		top := bottom - m.Rows
		for r := 0; r < m.Rows; r++ {
			for c := 0; c < m.Cols; c++ {
				src := c
				if parts[i].Flipped {
					src = m.Cols - 1 - c
				}
				tower.Set(c, top+r, m.At(src, r))
			}
		}
		parts[i].Row = top
		bottom = top
	}
	return tower, parts, nil
}

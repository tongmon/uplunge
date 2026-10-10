// Package level holds tile maps and loads level chunks. It must not depend on
// Ebitengine.
package level

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

// Tile is the content of one grid cell: the value of the chunk's Collision
// IntGrid layer. What a value means beyond Empty is defined by data (the
// block definitions in the tuning), applied to a map with SetShape.
type Tile uint8

const (
	Empty Tile = 0
	// Solid is the plain wall value every chunk uses.
	Solid Tile = 1
)

// Shape is how a tile collides.
type Shape uint8

const (
	// ShapeEmpty does not collide.
	ShapeEmpty Shape = iota
	// ShapeSolid blocks from every side.
	ShapeSolid
	// ShapeOneWay is a platform whose top edge blocks only bodies moving
	// down onto it.
	ShapeOneWay
)

// TileMap is a rectangular grid of tiles. Cell (0, 0) is the top-left, and y
// grows downward.
type TileMap struct {
	// Cols and Rows are the grid size in tiles.
	Cols, Rows int
	// TileSize is the edge length of one square tile in pixels.
	TileSize int

	tiles []Tile
	// shapes maps each tile value to its shape.
	shapes [256]Shape
}

// NewTileMap returns an all-empty map. Until SetShape says otherwise, Empty
// has ShapeEmpty and every other value ShapeSolid.
func NewTileMap(cols, rows, tileSize int) *TileMap {
	if cols < 0 || rows < 0 || tileSize <= 0 {
		panic(fmt.Sprintf("level: invalid tile map %dx%d with tile size %d", cols, rows, tileSize))
	}
	m := &TileMap{
		Cols:     cols,
		Rows:     rows,
		TileSize: tileSize,
		tiles:    make([]Tile, cols*rows),
	}
	for t := 1; t < len(m.shapes); t++ {
		m.shapes[t] = ShapeSolid
	}
	return m
}

// Clone returns an independent copy, so a world can break blocks without
// changing the chunk it was built from.
func (m *TileMap) Clone() *TileMap {
	c := *m
	c.tiles = append([]Tile(nil), m.tiles...)
	return &c
}

// SetShape sets how every tile of value t collides. Empty always stays
// ShapeEmpty.
func (m *TileMap) SetShape(t Tile, s Shape) {
	if t == Empty {
		return
	}
	m.shapes[t] = s
}

// ShapeAt returns how the tile at grid cell (col, row) collides. Cells outside
// the map are ShapeEmpty.
func (m *TileMap) ShapeAt(col, row int) Shape {
	return m.shapes[m.At(col, row)]
}

// Values returns the distinct non-empty tile values in the map, ascending.
func (m *TileMap) Values() []Tile {
	var seen [256]bool
	for _, t := range m.tiles {
		seen[t] = true
	}
	var vs []Tile
	for t := 1; t < len(seen); t++ {
		if seen[t] {
			vs = append(vs, Tile(t))
		}
	}
	return vs
}

// At returns the tile at grid cell (col, row). Cells outside the map are Empty.
func (m *TileMap) At(col, row int) Tile {
	if !m.inBounds(col, row) {
		return Empty
	}
	return m.tiles[row*m.Cols+col]
}

// Set changes the tile at grid cell (col, row), which must be inside the map.
func (m *TileMap) Set(col, row int, t Tile) {
	if !m.inBounds(col, row) {
		panic(fmt.Sprintf("level: cell (%d, %d) outside %dx%d map", col, row, m.Cols, m.Rows))
	}
	m.tiles[row*m.Cols+col] = t
}

func (m *TileMap) inBounds(col, row int) bool {
	return col >= 0 && col < m.Cols && row >= 0 && row < m.Rows
}

// Fingerprint returns a short hash of the size and tiles, so a replay can
// tell whether it is played back on the map it was recorded on.
func (m *TileMap) Fingerprint() string {
	h := sha256.New()
	var buf [8]byte
	for _, v := range []int{m.Cols, m.Rows, m.TileSize} {
		binary.LittleEndian.PutUint64(buf[:], uint64(v))
		h.Write(buf[:])
	}
	for _, t := range m.tiles {
		h.Write([]byte{byte(t)})
	}
	return hex.EncodeToString(h.Sum(nil)[:8])
}

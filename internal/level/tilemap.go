// Package level holds tile maps and loads level chunks. It must not depend on
// Ebitengine.
package level

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

// Tile is the collision kind of one grid cell.
type Tile uint8

const (
	Empty Tile = iota
	Solid
)

// TileMap is a rectangular grid of tiles. Cell (0, 0) is the top-left, and y
// grows downward.
type TileMap struct {
	// Cols and Rows are the grid size in tiles.
	Cols, Rows int
	// TileSize is the edge length of one square tile in pixels.
	TileSize int

	tiles []Tile
}

// NewTileMap returns an all-empty map.
func NewTileMap(cols, rows, tileSize int) *TileMap {
	if cols < 0 || rows < 0 || tileSize <= 0 {
		panic(fmt.Sprintf("level: invalid tile map %dx%d with tile size %d", cols, rows, tileSize))
	}
	return &TileMap{
		Cols:     cols,
		Rows:     rows,
		TileSize: tileSize,
		tiles:    make([]Tile, cols*rows),
	}
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

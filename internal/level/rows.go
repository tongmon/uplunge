package level

import "fmt"

// ParseRows builds a map from text rows, one character per tile: '#' is Solid
// and '.' is Empty. All rows must have the same length.
func ParseRows(tileSize int, rows ...string) (*TileMap, error) {
	if len(rows) == 0 {
		return nil, fmt.Errorf("level: no rows")
	}
	m := NewTileMap(len(rows[0]), len(rows), tileSize)
	for r, row := range rows {
		if len(row) != m.Cols {
			return nil, fmt.Errorf("level: row %d has %d cells, want %d", r, len(row), m.Cols)
		}
		for c := 0; c < len(row); c++ {
			switch row[c] {
			case '#':
				m.Set(c, r, Solid)
			case '.':
			default:
				return nil, fmt.Errorf("level: unknown cell %q at (%d, %d)", row[c], c, r)
			}
		}
	}
	return m, nil
}

package level

import "fmt"

// ParseRows builds a map from text rows, one character per tile: '#' is Solid,
// '.' is Empty, and a digit 2 to 9 is that tile value. All rows must have the
// same length.
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
			case '2', '3', '4', '5', '6', '7', '8', '9':
				m.Set(c, r, Tile(row[c]-'0'))
			default:
				return nil, fmt.Errorf("level: unknown cell %q at (%d, %d)", row[c], c, r)
			}
		}
	}
	return m, nil
}

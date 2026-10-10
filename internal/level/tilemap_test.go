package level

import "testing"

func TestTileMapAt(t *testing.T) {
	m := NewTileMap(3, 2, 16)
	m.Set(2, 1, Solid)

	tests := []struct {
		name     string
		col, row int
		want     Tile
	}{
		{"set cell", 2, 1, Solid},
		{"unset cell", 0, 0, Empty},
		{"left of map", -1, 0, Empty},
		{"right of map", 3, 1, Empty},
		{"above map", 2, -1, Empty},
		{"below map", 2, 2, Empty},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := m.At(tt.col, tt.row); got != tt.want {
				t.Fatalf("At(%d, %d) = %v, want %v", tt.col, tt.row, got, tt.want)
			}
		})
	}
}

func TestTileMapSetOutOfBoundsPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("Set outside the map did not panic")
		}
	}()
	NewTileMap(3, 2, 16).Set(3, 0, Solid)
}

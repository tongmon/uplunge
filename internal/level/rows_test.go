package level

import "testing"

func TestParseRows(t *testing.T) {
	m, err := ParseRows(16, "#.#", "..#")
	if err != nil {
		t.Fatal(err)
	}
	if m.Cols != 3 || m.Rows != 2 || m.TileSize != 16 {
		t.Fatalf("got %dx%d tile %d, want 3x2 tile 16", m.Cols, m.Rows, m.TileSize)
	}
	want := [][]Tile{{Solid, Empty, Solid}, {Empty, Empty, Solid}}
	for r := range want {
		for c := range want[r] {
			if got := m.At(c, r); got != want[r][c] {
				t.Errorf("At(%d, %d) = %v, want %v", c, r, got, want[r][c])
			}
		}
	}
}

func TestParseRowsErrors(t *testing.T) {
	tests := []struct {
		name string
		rows []string
	}{
		{"no rows", nil},
		{"ragged", []string{"##", "#"}},
		{"unknown cell", []string{"#x"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ParseRows(16, tt.rows...); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

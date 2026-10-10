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

func TestTileMapFingerprint(t *testing.T) {
	a, _ := ParseRows(16, "#..", "..#")
	b, _ := ParseRows(16, "#..", "..#")
	if a.Fingerprint() != b.Fingerprint() {
		t.Error("equal maps give different fingerprints")
	}
	changed, _ := ParseRows(16, "#..", ".##")
	transposed, _ := ParseRows(16, "#.", "..", ".#")
	bigger, _ := ParseRows(32, "#..", "..#")
	for name, m := range map[string]*TileMap{"tile": changed, "shape": transposed, "tile size": bigger} {
		if m.Fingerprint() == a.Fingerprint() {
			t.Errorf("different %s gives the same fingerprint", name)
		}
	}
}

func TestShapes(t *testing.T) {
	m := NewTileMap(4, 1, 16)
	m.Set(1, 0, Solid)
	m.Set(2, 0, 2)
	m.Set(3, 0, 3)
	m.SetShape(2, ShapeOneWay)
	m.SetShape(Empty, ShapeSolid) // ignored
	want := []Shape{ShapeEmpty, ShapeSolid, ShapeOneWay, ShapeSolid}
	for c, s := range want {
		if got := m.ShapeAt(c, 0); got != s {
			t.Errorf("ShapeAt(%d, 0) = %v, want %v", c, got, s)
		}
	}
	if got := m.ShapeAt(-1, 0); got != ShapeEmpty {
		t.Errorf("ShapeAt outside = %v, want ShapeEmpty", got)
	}
}

func TestCloneIsIndependent(t *testing.T) {
	m := NewTileMap(2, 1, 16)
	m.Set(0, 0, Solid)
	c := m.Clone()
	c.Set(0, 0, Empty)
	c.SetShape(Solid, ShapeOneWay)
	if m.At(0, 0) != Solid || m.ShapeAt(0, 0) != ShapeSolid {
		t.Fatal("changing the clone changed the original")
	}
	if m.Fingerprint() == c.Fingerprint() {
		t.Fatal("clone with a different tile has the same fingerprint")
	}
}

func TestValues(t *testing.T) {
	m, err := ParseRows(16, "#.3", "53.")
	if err != nil {
		t.Fatal(err)
	}
	got := m.Values()
	if len(got) != 3 || got[0] != Solid || got[1] != 3 || got[2] != 5 {
		t.Fatalf("Values() = %v, want [1 3 5]", got)
	}
}

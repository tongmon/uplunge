package collide

import (
	"testing"

	"github.com/tongmon/uplunge/internal/level"
)

const ts = 16

// mapFromRows builds a map from rows of '#' (solid) and '.' (empty).
func mapFromRows(t *testing.T, rows ...string) *level.TileMap {
	t.Helper()
	m := level.NewTileMap(len(rows[0]), len(rows), ts)
	for r, row := range rows {
		if len(row) != m.Cols {
			t.Fatalf("row %d has %d cells, want %d", r, len(row), m.Cols)
		}
		for c, ch := range row {
			switch ch {
			case '#':
				m.Set(c, r, level.Solid)
			case '.':
			default:
				t.Fatalf("unknown cell %q at (%d, %d)", ch, c, r)
			}
		}
	}
	return m
}

// testRoom has an open interior spanning x 16..79 and y 16..63, plus one solid
// tile at x 48..63, y 48..63.
func testRoom(t *testing.T) *level.TileMap {
	return mapFromRows(t,
		"######",
		"#....#",
		"#....#",
		"#..#.#",
		"######",
	)
}

func TestMove(t *testing.T) {
	const w, h = 10, 12
	tests := []struct {
		name         string
		vertical     bool
		x, y         int
		d            float64
		wantX, wantY int
		wantHit      bool
	}{
		{"free move right", false, 20, 20, 5, 25, 20, false},
		{"free move up", true, 20, 40, -5, 20, 35, false},
		{"stop at right wall", false, 20, 20, 100, 80 - w, 20, true},
		{"stop at left wall", false, 20, 20, -100, 16, 20, true},
		{"already touching wall", false, 80 - w, 20, 1, 80 - w, 20, true},
		{"fast move does not tunnel through one tile", false, 20, 50, 200, 48 - w, 50, true},
		{"land on floor", true, 20, 20, 100, 20, 64 - h, true},
		{"stop at ceiling", true, 20, 20, -100, 20, 16, true},
		{"land on block", true, 50, 20, 100, 50, 48 - h, true},
		{"straddling two columns lands on the solid one", true, 40, 20, 100, 40, 48 - h, true},
		{"edge-adjacent column falls past the block", true, 48 - w, 20, 100, 48 - w, 64 - h, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := testRoom(t)
			b := Body{X: tt.x, Y: tt.y, W: w, H: h}
			var hit bool
			if tt.vertical {
				hit = b.MoveY(m, tt.d)
			} else {
				hit = b.MoveX(m, tt.d)
			}
			if b.X != tt.wantX || b.Y != tt.wantY || hit != tt.wantHit {
				t.Fatalf("got (%d, %d) hit=%v, want (%d, %d) hit=%v",
					b.X, b.Y, hit, tt.wantX, tt.wantY, tt.wantHit)
			}
		})
	}
}

func TestMoveAccumulatesSubPixels(t *testing.T) {
	tests := []struct {
		name  string
		d     float64
		steps int
		want  int
	}{
		{"0.4 px for 5 steps", 0.4, 5, 22},
		{"-0.4 px for 5 steps", -0.4, 5, 18},
		{"0.25 px for 8 steps", 0.25, 8, 22},
		{"0.1 px for 4 steps stays put", 0.1, 4, 20},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := testRoom(t)
			b := Body{X: 20, Y: 20, W: 10, H: 12}
			for i := 0; i < tt.steps; i++ {
				b.MoveX(m, tt.d)
			}
			if b.X != tt.want {
				t.Fatalf("X = %d, want %d", b.X, tt.want)
			}
		})
	}
}

func TestHitDiscardsRemainder(t *testing.T) {
	m := testRoom(t)
	b := Body{X: 70, Y: 20, W: 10, H: 12}
	if b.MoveX(m, 0.4) {
		t.Fatal("sub-pixel move reported a hit")
	}
	if !b.MoveX(m, 0.4) {
		t.Fatal("move into wall did not report a hit")
	}
	if b.remX != 0 {
		t.Fatalf("remX = %v after hit, want 0", b.remX)
	}
}

func TestOverlaps(t *testing.T) {
	m := testRoom(t)
	tests := []struct {
		name       string
		x, y, w, h int
		want       bool
	}{
		{"inside open space", 16, 16, 32, 32, false},
		{"touching block edge from left", 38, 48, 10, 10, false},
		{"one pixel into block", 39, 48, 10, 10, true},
		{"touching floor edge", 20, 54, 10, 10, false},
		{"one pixel into floor", 20, 55, 10, 10, true},
		{"entirely outside map", -100, -100, 10, 10, false},
		{"straddling map edge into wall", -5, 20, 10, 10, true},
		{"zero size", 0, 0, 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Overlaps(m, tt.x, tt.y, tt.w, tt.h); got != tt.want {
				t.Fatalf("Overlaps(%d, %d, %d, %d) = %v, want %v",
					tt.x, tt.y, tt.w, tt.h, got, tt.want)
			}
		})
	}
}

func TestFloorDiv(t *testing.T) {
	tests := []struct{ a, b, want int }{
		{0, 16, 0},
		{15, 16, 0},
		{16, 16, 1},
		{-1, 16, -1},
		{-16, 16, -1},
		{-17, 16, -2},
	}
	for _, tt := range tests {
		if got := floorDiv(tt.a, tt.b); got != tt.want {
			t.Errorf("floorDiv(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

package collide

import (
	"testing"

	"github.com/tongmon/uplunge/internal/level"
)

// testRoom has 16 px tiles, an open interior spanning x 16..79 and y 16..63,
// and one solid tile at x 48..63, y 48..63.
func testRoom(t *testing.T) *level.TileMap {
	t.Helper()
	m, err := level.ParseRows(16,
		"######",
		"#....#",
		"#....#",
		"#..#.#",
		"######",
	)
	if err != nil {
		t.Fatal(err)
	}
	return m
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

// A remainder of exactly ±0.5 must not turn into movement once the body stops.
func TestHalfPixelRemainderHoldsStill(t *testing.T) {
	tests := []struct {
		name     string
		vertical bool
		d        float64
	}{
		{"x +0.5", false, 0.5},
		{"x -0.5", false, -0.5},
		{"y +0.5", true, 0.5},
		{"y -0.5", true, -0.5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := testRoom(t)
			b := Body{X: 30, Y: 30, W: 10, H: 12}
			move := b.MoveX
			if tt.vertical {
				move = b.MoveY
			}
			move(m, tt.d)
			wantX, wantY := b.X, b.Y
			for i := 0; i < 4; i++ {
				if move(m, 0) {
					t.Fatalf("step %d: zero move reported a hit", i)
				}
				if b.X != wantX || b.Y != wantY {
					t.Fatalf("step %d: zero move went to (%d, %d), want (%d, %d)",
						i, b.X, b.Y, wantX, wantY)
				}
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

func TestResizeKeepsFeetCentre(t *testing.T) {
	m := testRoom(t)
	b := Body{X: 20, Y: 40, W: 12, H: 20}
	for _, size := range [][2]int{{13, 20}, {14, 21}, {15, 18}, {12, 20}} {
		if !b.Resize(m, size[0], size[1]) {
			t.Fatalf("resize to %v blocked in open space", size)
		}
		if bottom := b.Y + b.H; bottom != 60 {
			t.Fatalf("after resize to %v the feet are at y %d, want 60", size, bottom)
		}
		// Centre in half pixels: 2*(left edge) + width, left edge = X + remX.
		if c2 := 2*(float64(b.X)+b.remX) + float64(b.W); c2 != 2*20+12 {
			t.Fatalf("after resize to %v the centre is %v half pixels, want %d", size, c2, 2*20+12)
		}
	}
	if b.X != 20 || b.remX != 0 {
		t.Fatalf("back at 12x20 the body is at x %d + %v, want 20 + 0", b.X, b.remX)
	}
}

func TestResizeBlocked(t *testing.T) {
	m := testRoom(t)
	b := Body{X: 20, Y: 44, W: 10, H: 20} // feet on the floor at y 64
	if b.Resize(m, 10, 60) {
		t.Fatal("resize through the ceiling succeeded")
	}
	if b != (Body{X: 20, Y: 44, W: 10, H: 20}) {
		t.Fatalf("blocked resize changed the body: %+v", b)
	}
}

// oneWayRoom has a one-way platform (value 2) whose top edge is at y 48,
// spanning x 16..63, over a floor at y 80.
func oneWayRoom(t *testing.T) *level.TileMap {
	t.Helper()
	m, err := level.ParseRows(16,
		"######",
		"#....#",
		"#....#",
		"#222.#",
		"#....#",
		"######",
	)
	if err != nil {
		t.Fatal(err)
	}
	m.SetShape(2, level.ShapeOneWay)
	return m
}

func TestOneWay(t *testing.T) {
	const w, h = 10, 12
	tests := []struct {
		name    string
		x, y    int
		dy      float64
		through bool
		wantY   int
		wantHit bool
	}{
		{"lands on the top edge", 20, 20, 100, false, 48 - h, true},
		{"already resting does not move", 20, 48 - h, 5, false, 48 - h, true},
		{"passes up through from below", 20, 60, -40, false, 20, false},
		{"falls on from inside the tile", 20, 50, 100, false, 80 - h, true},
		{"beside the platform falls to the floor", 64, 20, 100, false, 80 - h, true},
		{"through ignores the platform", 20, 20, 100, true, 80 - h, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := oneWayRoom(t)
			b := Body{X: tt.x, Y: tt.y, W: w, H: h}
			var hit bool
			if tt.through {
				hit = b.MoveYThrough(m, tt.dy)
			} else {
				hit = b.MoveY(m, tt.dy)
			}
			if b.Y != tt.wantY || hit != tt.wantHit {
				t.Fatalf("got y %d hit=%v, want y %d hit=%v", b.Y, hit, tt.wantY, tt.wantHit)
			}
		})
	}
}

func TestOneWayIsNotSolidSideways(t *testing.T) {
	m := oneWayRoom(t)
	b := Body{X: 70, Y: 50, W: 10, H: 12}
	if b.MoveX(m, -30); b.X != 40 {
		t.Fatalf("X = %d after moving left into the one-way tile, want 40", b.X)
	}
}

func TestOnGround(t *testing.T) {
	m := oneWayRoom(t)
	tests := []struct {
		name string
		x, y int
		want bool
	}{
		{"on the one-way top", 20, 48 - 12, true},
		{"one pixel above the one-way top", 20, 48 - 13, false},
		{"inside the one-way tile", 20, 40, false},
		{"on the floor", 64, 80 - 12, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := Body{X: tt.x, Y: tt.y, W: 10, H: 12}
			if got := b.OnGround(m); got != tt.want {
				t.Fatalf("OnGround = %v, want %v", got, tt.want)
			}
		})
	}
}

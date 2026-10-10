package level

import (
	"strings"
	"testing"

	"github.com/tongmon/uplunge/internal/rng"
)

func TestBuildLab(t *testing.T) {
	m, err := BuildLab(100, "E", 96, 16, rng.New(1))
	if err != nil {
		t.Fatal(err)
	}
	if m.Cols != ChunkCols || m.Rows != 100 {
		t.Fatalf("lab is %dx%d, want %dx100", m.Cols, m.Rows, ChunkCols)
	}
	for r := 0; r < 99; r++ {
		if m.At(0, r) != Solid || m.At(ChunkCols-1, r) != Solid || m.At(6, r) != Empty {
			t.Fatalf("row %d is not walls around open air", r)
		}
	}
	for c := 0; c < ChunkCols; c++ {
		if m.At(c, 99) != Solid {
			t.Fatal("no floor at the bottom")
		}
	}
	floor := 99 * 16
	if want := (floor - 2*16 - 1) / 96; len(m.Spawns) != want {
		t.Fatalf("%d spawns, want one every 96 px: %d", len(m.Spawns), want)
	}
	for i, s := range m.Spawns {
		base := floor - 96*(i+1)
		if s.Name != "E" || s.Y < base-16 || s.Y > base+16 || s.X < 24 || s.X > 12*16-8 {
			t.Fatalf("spawn %d = %+v, want E within 16 px of y %d between the walls", i, s, base)
		}
	}
	again, _ := BuildLab(100, "E", 96, 16, rng.New(1))
	if again.Fingerprint() != m.Fingerprint() {
		t.Fatal("the same seed built a different lab")
	}
}

func TestBuildLabErrors(t *testing.T) {
	for _, tt := range []struct {
		rows, spacing, jitter int
		wantErr               string
	}{
		{3, 96, 0, "at least 4 rows"},
		{100, 0, 0, "spacing must be positive"},
		{100, 96, 48, "jitter must be"},
		{100, 96, -1, "jitter must be"},
	} {
		if _, err := BuildLab(tt.rows, "E", tt.spacing, tt.jitter, rng.New(0)); err == nil || !strings.Contains(err.Error(), tt.wantErr) {
			t.Errorf("BuildLab(%d, %d, %d) error = %v, want %q", tt.rows, tt.spacing, tt.jitter, err, tt.wantErr)
		}
	}
}

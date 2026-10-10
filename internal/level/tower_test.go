package level

import (
	"strings"
	"testing"

	"github.com/tongmon/uplunge/internal/rng"
)

func towerChunks(t *testing.T) []Chunk {
	t.Helper()
	mk := func(rows ...string) *TileMap {
		m, err := ParseRows(16, rows...)
		if err != nil {
			t.Fatal(err)
		}
		return m
	}
	return []Chunk{
		{"Base", mk("#...#", "#####")},
		{"A", mk("#2..#")},
		{"B", mk("#.3.#", "#..3#")},
		{"C", mk("#...#", "#4..#", "#...#")},
	}
}

func TestBuildTowerStacksFromTheBottom(t *testing.T) {
	chunks := towerChunks(t)
	m, parts, err := BuildTower(chunks, "Base", []string{"A", "B", "C"}, 6, rng.New(7))
	if err != nil {
		t.Fatal(err)
	}
	if parts[0].Name != "Base" || parts[0].Flipped {
		t.Fatalf("bottom chunk %+v, want Base unflipped", parts[0])
	}
	rows := 0
	for i, p := range parts {
		if i > 0 && p.Name == parts[i-1].Name {
			t.Fatalf("chunk %q twice in a row at %d: %+v", p.Name, i, parts)
		}
		src, err := FindChunk(chunks, p.Name)
		if err != nil {
			t.Fatal(err)
		}
		rows += src.Rows
		// The tower holds each chunk at its row, mirrored when flipped.
		for r := 0; r < src.Rows; r++ {
			for c := 0; c < src.Cols; c++ {
				sc := c
				if p.Flipped {
					sc = src.Cols - 1 - c
				}
				if got, want := m.At(c, p.Row+r), src.At(sc, r); got != want {
					t.Fatalf("%s at tower (%d, %d) = %d, want %d", p.Name, c, p.Row+r, got, want)
				}
			}
		}
	}
	if m.Rows != rows || m.Cols != 5 {
		t.Fatalf("tower is %dx%d, want 5x%d", m.Cols, m.Rows, rows)
	}
	if last := parts[0]; last.Row+2 != m.Rows {
		t.Fatalf("base starts at row %d, want it at the bottom (%d)", last.Row, m.Rows-2)
	}
}

func TestBuildTowerIsDeterministic(t *testing.T) {
	chunks := towerChunks(t)
	pool := []string{"A", "B", "C"}
	a, pa, _ := BuildTower(chunks, "Base", pool, 8, rng.New(3))
	b, pb, _ := BuildTower(chunks, "Base", pool, 8, rng.New(3))
	if a.Fingerprint() != b.Fingerprint() {
		t.Fatal("same seed built different towers")
	}
	for i := range pa {
		if pa[i] != pb[i] {
			t.Fatalf("placement %d differs: %+v vs %+v", i, pa[i], pb[i])
		}
	}
	flips, names := 0, map[string]bool{}
	for seed := uint64(0); seed < 20; seed++ {
		_, ps, _ := BuildTower(chunks, "Base", pool, 8, rng.New(seed))
		for _, p := range ps[1:] {
			names[p.Name] = true
			if p.Flipped {
				flips++
			}
		}
	}
	if len(names) != 3 || flips == 0 {
		t.Fatalf("20 seeds used chunks %v with %d flips, want all pool chunks and some flips", names, flips)
	}
}

func TestBuildTowerErrors(t *testing.T) {
	chunks := towerChunks(t)
	tests := []struct {
		name    string
		base    string
		pool    []string
		length  int
		wantErr string
	}{
		{"zero length", "Base", []string{"A"}, 0, "must be 1 to at most"},
		{"unknown base", "Nope", []string{"A"}, 2, `no chunk "Nope"`},
		{"unknown pool chunk", "Base", []string{"A", "Nope"}, 2, `no chunk "Nope"`},
		{"empty pool", "Base", nil, 2, "needs a pool"},
		{"pool of only the base", "Base", []string{"Base"}, 2, "needs a pool"},
		{"one pool chunk for three", "Base", []string{"A"}, 3, "at least 2 distinct pool chunks"},
		{"one pool chunk listed twice", "Base", []string{"A", "A"}, 3, "at least 2 distinct pool chunks"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := BuildTower(chunks, tt.base, tt.pool, tt.length, rng.New(0))
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
	// The base alone needs no pool.
	if _, _, err := BuildTower(chunks, "Base", nil, 1, rng.New(0)); err != nil {
		t.Fatalf("base-only tower: %v", err)
	}
}

func TestBuildTowerRejectsHugeLength(t *testing.T) {
	_, _, err := BuildTower(towerChunks(t), "Base", []string{"A", "B"}, MaxTowerLength+1, rng.New(0))
	if err == nil || !strings.Contains(err.Error(), "at most") {
		t.Fatalf("error = %v, want one about the length limit", err)
	}
}

func TestBuildTowerPlacesSpawns(t *testing.T) {
	chunks := towerChunks(t)
	chunks[0].Map.Spawns = []Spawn{{"E", 10, 8}} // Base: 5x2 tiles
	chunks[1].Map.Spawns = []Spawn{{"E", 20, 4}} // A: 5x1 tiles
	m, parts, err := BuildTower(chunks, "Base", []string{"A"}, 2, rng.New(0))
	if err != nil {
		t.Fatal(err)
	}
	if parts[1].Name != "A" {
		t.Fatalf("second chunk %s, want A", parts[1].Name)
	}
	ax := 20
	if parts[1].Flipped {
		ax = 5*16 - 20
	}
	want := []Spawn{{"E", 10, 16 + 8}, {"E", ax, 4}}
	if len(m.Spawns) != 2 || m.Spawns[0] != want[0] || m.Spawns[1] != want[1] {
		t.Fatalf("spawns %+v, want %+v", m.Spawns, want)
	}
	// Mirroring maps a centre x to the mirrored centre.
	sawFlip := false
	for seed := uint64(0); seed < 10; seed++ {
		m, parts, _ := BuildTower(chunks, "Base", []string{"A"}, 2, rng.New(seed))
		if parts[1].Flipped {
			sawFlip = true
			if m.Spawns[1].X != 60 {
				t.Fatalf("flipped spawn x = %d, want 60", m.Spawns[1].X)
			}
		}
	}
	if !sawFlip {
		t.Fatal("no seed flipped the chunk")
	}
}

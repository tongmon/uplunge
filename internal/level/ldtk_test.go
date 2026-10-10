package level

import (
	"fmt"
	"strings"
	"testing"
)

// ldtkJSON builds a one-level project holding the given layer instances.
func ldtkJSON(layers string) string {
	return `{"externalLevels": false, "levels": [{"identifier": "A", "layerInstances": [` + layers + `]}]}`
}

// collisionJSON builds a Collision layer instance from '#'/'.' rows. extra is
// spliced in as additional fields.
func collisionJSON(gridSize int, extra string, rows ...string) string {
	var csv []string
	for _, r := range rows {
		for _, c := range r {
			if c == '#' {
				csv = append(csv, "1")
			} else {
				csv = append(csv, "0")
			}
		}
	}
	return fmt.Sprintf(`{"__identifier": "Collision", "__type": "IntGrid", %s
		"__cWid": %d, "__cHei": %d, "__gridSize": %d, "intGridCsv": [%s]}`,
		extra, len(rows[0]), len(rows), gridSize, strings.Join(csv, ", "))
}

var validRows = []string{
	"#.#..........",
	"............#",
}

func TestParseLDtk(t *testing.T) {
	other := `{"__identifier": "Decor", "__type": "Tiles", "__cWid": 1, "__cHei": 1, "__gridSize": 16}`
	zeroOffset := `"__pxTotalOffsetX": 0, "__pxTotalOffsetY": 0,`
	chunks, err := ParseLDtk([]byte(ldtkJSON(other + ", " + collisionJSON(16, zeroOffset, validRows...))))
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 1 || chunks[0].Name != "A" {
		t.Fatalf("got %+v, want one chunk named A", chunks)
	}
	m := chunks[0].Map
	if m.Cols != 13 || m.Rows != 2 || m.TileSize != 16 {
		t.Fatalf("got %dx%d tile %d, want 13x2 tile 16", m.Cols, m.Rows, m.TileSize)
	}
	want, err := ParseRows(16, validRows...)
	if err != nil {
		t.Fatal(err)
	}
	for r := 0; r < want.Rows; r++ {
		for c := 0; c < want.Cols; c++ {
			if got := m.At(c, r); got != want.At(c, r) {
				t.Errorf("At(%d, %d) = %v, want %v", c, r, got, want.At(c, r))
			}
		}
	}
}

func TestParseLDtkErrors(t *testing.T) {
	valid := collisionJSON(16, "", validRows...)
	tests := []struct {
		name    string
		json    string
		wantErr string
	}{
		{"not json", `{`, "unexpected end"},
		{"external levels", `{"externalLevels": true, "levels": []}`, "separate files"},
		{"no levels", `{"levels": []}`, "no levels"},
		{"multi-world", `{"levels": [], "worlds": [{"levels": []}]}`, "multi-world"},
		{"null layer instances", `{"levels": [{"identifier": "A", "layerInstances": null}]}`, "no layer instances"},
		{"no collision layer", ldtkJSON(`{"__identifier": "Decor", "__type": "IntGrid"}`), `no "Collision" layer`},
		{"wrong layer type", ldtkJSON(strings.Replace(valid, `"IntGrid"`, `"Tiles"`, 1)), "want IntGrid"},
		{"no rows", ldtkJSON(strings.Replace(valid, `"__cHei": 2`, `"__cHei": 0`, 1)), "no rows"},
		{"wrong width", ldtkJSON(collisionJSON(16, "", "#.#", "..#")), "3 tiles across, want 13"},
		{"wrong grid size", ldtkJSON(collisionJSON(8, "", validRows...)), "8 px tiles, want 16"},
		{"layer offset", ldtkJSON(collisionJSON(16, `"__pxTotalOffsetX": 16, "__pxTotalOffsetY": 0,`, validRows...)), "offset (16, 0)"},
		{"cell count mismatch", ldtkJSON(strings.Replace(valid, `"__cHei": 2`, `"__cHei": 3`, 1)), "has 26 cells, want 39"},
		{"value out of range", ldtkJSON(strings.Replace(valid, `"intGridCsv": [1, 0`, `"intGridCsv": [1, 256`, 1)), "value 256 at (1, 0) is outside 0..255"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseLDtk([]byte(tt.json))
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestShippedChunks(t *testing.T) {
	if _, err := LoadLDtk("../../assets/chunks/chunks.ldtk"); err != nil {
		t.Fatal(err)
	}
}

func TestFindChunk(t *testing.T) {
	a, b := NewTileMap(1, 1, 16), NewTileMap(1, 1, 16)
	chunks := []Chunk{{"A", a}, {"B", b}}
	if m, err := FindChunk(chunks, "B"); err != nil || m != b {
		t.Fatalf("FindChunk(B) = %p, %v; want %p", m, err, b)
	}
	_, err := FindChunk(chunks, "C")
	if err == nil || !strings.Contains(err.Error(), "have A, B") {
		t.Fatalf("FindChunk(C) error = %v, want it to list the chunks", err)
	}
}

func TestParseLDtkEntities(t *testing.T) {
	entities := `{"__identifier": "Entities", "__type": "Entities", "__cWid": 13, "__cHei": 2, "__gridSize": 16,
		"entityInstances": [
			{"__identifier": "Floater", "px": [32, 16], "__pivot": [0, 0], "width": 16, "height": 16},
			{"__identifier": "Spiker", "px": [104, 32], "__pivot": [0.5, 1], "width": 16, "height": 16}
		]}`
	chunks, err := ParseLDtk([]byte(ldtkJSON(entities + ", " + collisionJSON(16, "", validRows...))))
	if err != nil {
		t.Fatal(err)
	}
	want := []Spawn{{"Floater", 40, 24}, {"Spiker", 104, 24}}
	got := chunks[0].Map.Spawns
	if len(got) != len(want) {
		t.Fatalf("spawns %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("spawn %d = %+v, want %+v", i, got[i], want[i])
		}
	}
	bad := strings.Replace(entities, `"__type": "Entities"`, `"__type": "IntGrid"`, 1)
	if _, err := ParseLDtk([]byte(ldtkJSON(bad + ", " + collisionJSON(16, "", validRows...)))); err == nil ||
		!strings.Contains(err.Error(), "want Entities") {
		t.Fatalf("error = %v, want one about the layer type", err)
	}
}

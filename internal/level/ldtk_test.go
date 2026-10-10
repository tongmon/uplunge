package level

import (
	"strings"
	"testing"
)

// ldtkJSON builds a one-level project holding the given layer instances.
func ldtkJSON(layers string) string {
	return `{"externalLevels": false, "levels": [{"identifier": "A", "layerInstances": [` + layers + `]}]}`
}

const collision2x3 = `{"__identifier": "Collision", "__type": "IntGrid",
	"__cWid": 3, "__cHei": 2, "__gridSize": 16, "intGridCsv": [1, 0, 1, 0, 0, 1]}`

func TestParseLDtk(t *testing.T) {
	other := `{"__identifier": "Decor", "__type": "Tiles", "__cWid": 1, "__cHei": 1, "__gridSize": 16}`
	chunks, err := ParseLDtk([]byte(ldtkJSON(other + ", " + collision2x3)))
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 1 || chunks[0].Name != "A" {
		t.Fatalf("got %+v, want one chunk named A", chunks)
	}
	m := chunks[0].Map
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

func TestParseLDtkErrors(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr string
	}{
		{"not json", `{`, "unexpected end"},
		{"external levels", `{"externalLevels": true, "levels": []}`, "separate files"},
		{"no levels", `{"levels": []}`, "no levels"},
		{"null layer instances", `{"levels": [{"identifier": "A", "layerInstances": null}]}`, "no layer instances"},
		{"no collision layer", ldtkJSON(`{"__identifier": "Decor", "__type": "IntGrid"}`), `no "Collision" layer`},
		{"wrong layer type", ldtkJSON(strings.Replace(collision2x3, `"IntGrid"`, `"Tiles"`, 1)), "want IntGrid"},
		{"bad size", ldtkJSON(strings.Replace(collision2x3, `"__cWid": 3`, `"__cWid": 0`, 1)), "invalid size"},
		{"cell count mismatch", ldtkJSON(strings.Replace(collision2x3, `1, 0, 1, 0, 0, 1`, `1, 0, 1`, 1)), "has 3 cells, want 6"},
		{"unknown value", ldtkJSON(strings.Replace(collision2x3, `1, 0, 1, 0, 0, 1`, `1, 0, 1, 0, 7, 1`, 1)), "unknown \"Collision\" value 7 at (1, 1)"},
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

// Chunk authoring rules (design.md section 7): 16 px tiles, 13 tiles across
// including both walls.
const (
	chunkTileSize = 16
	chunkCols     = 13
)

func TestShippedChunks(t *testing.T) {
	chunks, err := LoadLDtk("../../assets/chunks/chunks.ldtk")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range chunks {
		if c.Map.TileSize != chunkTileSize || c.Map.Cols != chunkCols {
			t.Errorf("chunk %q is %d tiles of %d px across, want %d of %d px",
				c.Name, c.Map.Cols, c.Map.TileSize, chunkCols, chunkTileSize)
		}
	}
}

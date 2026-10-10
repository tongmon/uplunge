package level

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// CollisionLayer is the IntGrid layer every chunk must have. Its values map to
// tiles: 0 is Empty and 1 is Solid.
const CollisionLayer = "Collision"

// Chunk authoring rules (docs/design.md section 7): 16 px tiles, 13 tiles
// across including both walls. Height is free.
const (
	ChunkTileSize = 16
	ChunkCols     = 13
)

// Chunk is one hand-authored level piece: one level in the LDtk project.
type Chunk struct {
	Name string
	Map  *TileMap
}

// LoadLDtk reads every level of an LDtk project as a chunk.
func LoadLDtk(path string) ([]Chunk, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("level: %w", err)
	}
	chunks, err := ParseLDtk(data)
	if err != nil {
		return nil, fmt.Errorf("level: %s: %w", path, err)
	}
	return chunks, nil
}

// The subset of the LDtk project JSON the loader reads. LDtk writes many more
// fields; they are ignored.
type ldtkProject struct {
	ExternalLevels bool              `json:"externalLevels"`
	Levels         []ldtkLevel       `json:"levels"`
	Worlds         []json.RawMessage `json:"worlds"`
}

type ldtkLevel struct {
	Identifier     string       `json:"identifier"`
	LayerInstances *[]ldtkLayer `json:"layerInstances"`
}

type ldtkLayer struct {
	Identifier string `json:"__identifier"`
	Type       string `json:"__type"`
	Cols       int    `json:"__cWid"`
	Rows       int    `json:"__cHei"`
	GridSize   int    `json:"__gridSize"`
	IntGridCSV []int  `json:"intGridCsv"`
	// Sum of the layer definition's and this instance's pixel offsets.
	OffsetX int `json:"__pxTotalOffsetX"`
	OffsetY int `json:"__pxTotalOffsetY"`
}

// ParseLDtk decodes an LDtk project into chunks, in the project's level order.
func ParseLDtk(data []byte) ([]Chunk, error) {
	var p ldtkProject
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	if p.ExternalLevels {
		return nil, fmt.Errorf("levels saved in separate files are not supported")
	}
	if len(p.Levels) == 0 && len(p.Worlds) > 0 {
		return nil, fmt.Errorf("multi-world projects are not supported")
	}
	if len(p.Levels) == 0 {
		return nil, fmt.Errorf("project has no levels")
	}
	chunks := make([]Chunk, 0, len(p.Levels))
	for _, lv := range p.Levels {
		m, err := parseLevel(lv)
		if err != nil {
			return nil, fmt.Errorf("level %q: %w", lv.Identifier, err)
		}
		chunks = append(chunks, Chunk{Name: lv.Identifier, Map: m})
	}
	return chunks, nil
}

func parseLevel(lv ldtkLevel) (*TileMap, error) {
	if lv.LayerInstances == nil {
		return nil, fmt.Errorf("no layer instances")
	}
	var layer *ldtkLayer
	for i := range *lv.LayerInstances {
		if l := &(*lv.LayerInstances)[i]; l.Identifier == CollisionLayer {
			layer = l
			break
		}
	}
	if layer == nil {
		return nil, fmt.Errorf("no %q layer", CollisionLayer)
	}
	if layer.Type != "IntGrid" {
		return nil, fmt.Errorf("layer %q is %s, want IntGrid", CollisionLayer, layer.Type)
	}
	if layer.GridSize != ChunkTileSize {
		return nil, fmt.Errorf("layer %q has %d px tiles, want %d", CollisionLayer, layer.GridSize, ChunkTileSize)
	}
	if layer.Cols != ChunkCols {
		return nil, fmt.Errorf("layer %q is %d tiles across, want %d", CollisionLayer, layer.Cols, ChunkCols)
	}
	if layer.Rows <= 0 {
		return nil, fmt.Errorf("layer %q has no rows", CollisionLayer)
	}
	if layer.OffsetX != 0 || layer.OffsetY != 0 {
		return nil, fmt.Errorf("layer %q has pixel offset (%d, %d), want none",
			CollisionLayer, layer.OffsetX, layer.OffsetY)
	}
	if got, want := len(layer.IntGridCSV), layer.Cols*layer.Rows; got != want {
		return nil, fmt.Errorf("layer %q has %d cells, want %d", CollisionLayer, got, want)
	}

	m := NewTileMap(layer.Cols, layer.Rows, layer.GridSize)
	for i, v := range layer.IntGridCSV {
		col, row := i%layer.Cols, i/layer.Cols
		switch v {
		case 0:
		case 1:
			m.Set(col, row, Solid)
		default:
			return nil, fmt.Errorf("unknown %q value %d at (%d, %d)", CollisionLayer, v, col, row)
		}
	}
	return m, nil
}

// FindChunk returns the map of the chunk called name.
func FindChunk(chunks []Chunk, name string) (*TileMap, error) {
	names := make([]string, len(chunks))
	for i, c := range chunks {
		if c.Name == name {
			return c.Map, nil
		}
		names[i] = c.Name
	}
	return nil, fmt.Errorf("level: no chunk %q; have %s", name, strings.Join(names, ", "))
}

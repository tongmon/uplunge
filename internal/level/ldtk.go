package level

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
)

// CollisionLayer is the IntGrid layer every chunk must have. Its values are
// the tiles: 0 is Empty, 1 is Solid, and the block definitions in the tuning
// give every other value its meaning.
const CollisionLayer = "Collision"

// EntitiesLayer is the optional Entities layer that places enemies and other
// entities in a chunk.
const EntitiesLayer = "Entities"

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
	Defs           struct {
		Layers   []ldtkLayerDef `json:"layers"`
		Entities []struct {
			Identifier string `json:"identifier"`
		} `json:"entities"`
	} `json:"defs"`
}

type ldtkLayerDef struct {
	Identifier    string         `json:"identifier"`
	IntGridValues []IntGridValue `json:"intGridValues"`
}

// IntGridValue is one value an LDtk IntGrid layer defines, with the name the
// editor shows for it.
type IntGridValue struct {
	Value int    `json:"value"`
	Name  string `json:"identifier"`
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

	Entities []ldtkEntity `json:"entityInstances"`
}

type ldtkEntity struct {
	Identifier string `json:"__identifier"`
	// Px is the entity's pivot point; Pivot is where that point sits in the
	// entity, as fractions of its size.
	Px     [2]int     `json:"px"`
	Pivot  [2]float64 `json:"__pivot"`
	Width  int        `json:"width"`
	Height int        `json:"height"`
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
	var layer, entities *ldtkLayer
	for i := range *lv.LayerInstances {
		switch l := &(*lv.LayerInstances)[i]; l.Identifier {
		case CollisionLayer:
			layer = l
		case EntitiesLayer:
			entities = l
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
		if v < 0 || v > 255 {
			return nil, fmt.Errorf("%q value %d at (%d, %d) is outside 0..255", CollisionLayer, v, col, row)
		}
		m.Set(col, row, Tile(v))
	}
	if entities != nil {
		if entities.Type != "Entities" {
			return nil, fmt.Errorf("layer %q is %s, want Entities", EntitiesLayer, entities.Type)
		}
		if entities.OffsetX != 0 || entities.OffsetY != 0 {
			return nil, fmt.Errorf("layer %q has pixel offset (%d, %d), want none",
				EntitiesLayer, entities.OffsetX, entities.OffsetY)
		}
		for _, e := range entities.Entities {
			left := e.Px[0] - int(math.Round(e.Pivot[0]*float64(e.Width)))
			top := e.Px[1] - int(math.Round(e.Pivot[1]*float64(e.Height)))
			m.Spawns = append(m.Spawns, Spawn{Name: e.Identifier, X: left + e.Width/2, Y: top + e.Height/2})
		}
	}
	return m, nil
}

// CollisionValues returns the values the project defines for its Collision
// layer, so they can be checked against the block definitions.
func CollisionValues(path string) ([]IntGridValue, error) {
	p, err := loadProject(path)
	if err != nil {
		return nil, err
	}
	for _, l := range p.Defs.Layers {
		if l.Identifier == CollisionLayer {
			return l.IntGridValues, nil
		}
	}
	return nil, fmt.Errorf("level: %s: no %q layer definition", path, CollisionLayer)
}

// EntityNames returns the identifiers of the entities the project defines,
// so they can be checked against the enemy definitions.
func EntityNames(path string) ([]string, error) {
	p, err := loadProject(path)
	if err != nil {
		return nil, err
	}
	names := make([]string, len(p.Defs.Entities))
	for i, e := range p.Defs.Entities {
		names[i] = e.Identifier
	}
	return names, nil
}

func loadProject(path string) (ldtkProject, error) {
	var p ldtkProject
	data, err := os.ReadFile(path)
	if err != nil {
		return p, fmt.Errorf("level: %w", err)
	}
	if err := json.Unmarshal(data, &p); err != nil {
		return p, fmt.Errorf("level: %s: %w", path, err)
	}
	return p, nil
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

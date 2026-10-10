package sim

import (
	"fmt"

	"github.com/tongmon/uplunge/internal/collide"
	"github.com/tongmon/uplunge/internal/level"
	"github.com/tongmon/uplunge/internal/tuning"
)

// blockTable maps each tile value to its block definition. An entry whose
// Value is 0 is undefined.
type blockTable [256]tuning.Block

func newBlockTable(blocks []tuning.Block) *blockTable {
	var bt blockTable
	for _, b := range blocks {
		bt[b.Value] = b
	}
	return &bt
}

// check reports the first tile value in m that has no definition.
func (bt *blockTable) check(m *level.TileMap) error {
	for _, v := range m.Values() {
		if bt[v].Value == 0 {
			return fmt.Errorf("sim: tile value %d has no block definition", v)
		}
	}
	return nil
}

// apply gives every defined value of m its collision shape.
func (bt *blockTable) apply(m *level.TileMap) {
	for v := 1; v < len(bt); v++ {
		if bt[v].Value == 0 {
			continue
		}
		s := level.ShapeSolid
		if bt[v].OneWay {
			s = level.ShapeOneWay
		}
		m.SetShape(level.Tile(v), s)
	}
}

// breakIn empties every tile the box at (x, y) of size w×h overlaps whose
// block breaks by the given cause, records each in ev, and reports whether
// any broke. byBullet says the cause, for the record.
func (bt *blockTable) breakIn(m *level.TileMap, x, y, w, h int, breaks func(tuning.Block) bool, byBullet bool, ev *Events) bool {
	broke := false
	collide.EachTile(m, x, y, w, h, func(c, r int) bool {
		if t := m.At(c, r); t != level.Empty && breaks(bt[t]) {
			m.Set(c, r, level.Empty)
			ev.addBroken(BrokenCell{Col: c, Row: r, Tile: t, ByBullet: byBullet})
			broke = true
		}
		return true
	})
	return broke
}

func byDrill(b tuning.Block) bool  { return b.Drill }
func byBullet(b tuning.Block) bool { return b.Bullet }

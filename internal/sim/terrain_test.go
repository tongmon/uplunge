package sim

import (
	"strings"
	"testing"

	"github.com/tongmon/uplunge/internal/level"
	"github.com/tongmon/uplunge/internal/tuning"
)

// Block values of testTuning.
const (
	oneWay     level.Tile = 2
	drill      level.Tile = 3
	soft       level.Tile = 4
	bulletOnly level.Tile = 5
)

// roomWithRow returns testRoom (floor top at y 176) with row r filled with
// value v between the walls, and the player standing on the floor at x.
func roomWithRow(t *testing.T, r int, v level.Tile, x int) *World {
	t.Helper()
	m := testRoom(t)
	for c := 1; c < 12; c++ {
		m.Set(c, r, v)
	}
	w := NewWorld(testTuning(), m, x, floorY-testTuning().Player.Height)
	w.Step(Input{})
	if !w.Player.OnGround {
		t.Fatal("player does not start on the floor")
	}
	return w
}

func TestJumpThroughOneWayAndLandOnIt(t *testing.T) {
	// The platform top is 48 px above the floor, within a held jump.
	w := roomWithRow(t, 8, oneWay, 96)
	run(w, Input{Button: true}, 12)
	run(w, Input{}, Hz)
	p := w.Player
	if !p.OnGround || p.Body.Y+p.Body.H != 8*tile {
		t.Fatalf("feet at %d OnGround=%v, want standing on the platform top at %d",
			p.Body.Y+p.Body.H, p.OnGround, 8*tile)
	}
}

func TestStandingOnOneWayRefills(t *testing.T) {
	w := roomWithRow(t, 8, oneWay, 96)
	run(w, Input{Button: true}, 12)
	run(w, Input{}, Hz)
	w.Player.Fuel = 0
	w.Step(Input{})
	if w.Player.Fuel != testTuning().Gun.Magazine {
		t.Fatalf("Fuel = %d on a one-way platform, want refilled", w.Player.Fuel)
	}
}

// landsOnPlatform jumps holding the button for hold steps under a one-way
// platform whose top is 48 px above the floor, and reports whether the
// player ends up standing on it.
func landsOnPlatform(t *testing.T, tun tuning.Tuning, hold int) bool {
	t.Helper()
	m := testRoom(t)
	for c := 1; c < 12; c++ {
		m.Set(c, 8, oneWay)
	}
	w := NewWorld(tun, m, 96, floorY-tun.Player.Height)
	w.Step(Input{})
	for i := 0; i < 2*Hz; i++ {
		w.Step(Input{Button: i < hold})
	}
	return w.Player.OnGround && w.Player.Body.Y+w.Player.Body.H == 8*tile
}

func TestOneWayAssistLiftsAShortJumpOnTop(t *testing.T) {
	weak := testTuning()
	weak.Player.OneWayAssist = 0.001
	helped := false
	for hold := 1; hold <= 12; hold++ {
		with, without := landsOnPlatform(t, testTuning(), hold), landsOnPlatform(t, weak, hold)
		if without && !with {
			t.Fatalf("hold %d lands without the assist but not with it", hold)
		}
		if with && !without {
			helped = true
		}
	}
	if !helped {
		t.Fatal("the assist never turned a short jump into a landing")
	}
}

func TestBulletsPassThroughOneWay(t *testing.T) {
	w := tallWorld()
	w.Map.Set(6, 103, oneWay) // 2 px under the player's feet at y 1620
	w.blocks.apply(w.Map)
	w.Step(Input{})
	w.Step(Input{Button: true})
	run(w, Input{}, 6)
	if len(w.Bullets) != 1 || w.Bullets[0].Body.Y < 104*tile {
		t.Fatalf("bullets %+v, want one bullet past the platform", w.Bullets)
	}
	if w.Map.At(6, 103) != oneWay {
		t.Fatal("bullet broke the one-way platform")
	}
}

// drillCeiling jumps from the floor into a row-8 ceiling (12 px above the
// head) of value v and returns the world once the jump is over.
func drillCeiling(t *testing.T, v level.Tile) (w *World, vyAtHit float64) {
	t.Helper()
	w = roomWithRow(t, 8, v, 96)
	for i := 0; i < 12; i++ {
		w.Step(Input{Button: true})
		if w.Map.At(6, 8) == level.Empty && vyAtHit == 0 {
			vyAtHit = w.Player.VY
		}
	}
	return w, vyAtHit
}

func TestHeadBreaksDrillBlocks(t *testing.T) {
	for _, v := range []level.Tile{drill, soft} {
		w, vy := drillCeiling(t, v)
		if w.Map.At(6, 8) != level.Empty {
			t.Fatalf("value %d: block above the head not broken", v)
		}
		if w.Map.At(5, 8) != v || w.Map.At(7, 8) != v {
			t.Fatalf("value %d: blocks beside the head broke too", v)
		}
		if vy != -testTuning().Player.DrillBounce {
			t.Fatalf("value %d: VY = %v on the breaking step, want -DrillBounce", v, vy)
		}
		top := w.Player.Body.Y
		run(w, Input{}, Hz/4)
		top = min(top, w.Player.Body.Y)
		if top >= 8*tile {
			t.Fatalf("value %d: head top reached %d, want through the hole above %d", v, top, 8*tile)
		}
	}
}

func TestHeadDoesNotBreakOtherBlocks(t *testing.T) {
	for _, v := range []level.Tile{level.Solid, bulletOnly} {
		w, _ := drillCeiling(t, v)
		if w.Map.At(6, 8) != v {
			t.Fatalf("value %d broke under the head", v)
		}
		if w.Player.Body.Y < 9*tile {
			t.Fatalf("value %d: player passed the ceiling", v)
		}
	}
}

// shootDown fires one bullet from 20 px above row 105 of a tall world whose
// row 105 is filled with v, and returns the world once the bullet is gone.
func shootDown(t *testing.T, v level.Tile) *World {
	t.Helper()
	w := tallWorld() // feet at y 1620, row 102 bottom
	for c := 1; c < 12; c++ {
		w.Map.Set(c, 104, v) // top at y 1664
	}
	w.blocks.apply(w.Map)
	w.Step(Input{})
	w.Step(Input{Button: true})
	for i := 0; i < 20 && len(w.Bullets) > 0; i++ {
		w.Step(Input{})
	}
	if len(w.Bullets) != 0 {
		t.Fatal("bullet still flying")
	}
	return w
}

func TestBulletBreaksBulletBlocks(t *testing.T) {
	tests := []struct {
		v      level.Tile
		break_ bool
	}{
		{soft, true},
		{bulletOnly, true},
		{drill, false},
		{level.Solid, false},
	}
	for _, tt := range tests {
		w := shootDown(t, tt.v)
		got := w.Map.At(6, 104) == level.Empty
		if got != tt.break_ {
			t.Fatalf("value %d: broke = %v, want %v", tt.v, got, tt.break_)
		}
		if w.Map.At(5, 104) != tt.v || w.Map.At(7, 104) != tt.v {
			t.Fatalf("value %d: blocks beside the bullet broke too", tt.v)
		}
	}
}

func TestBulletSpawnedInsideABlockBreaksIt(t *testing.T) {
	// Feet 2 px above a soft block. The shot lifts the player 4 px, and the
	// 8 px bullet spawned at the feet still reaches 2 px into the block.
	const r = 104
	m := level.NewTileMap(13, 200, tile)
	m.Set(6, r, soft)
	w := NewWorld(testTuning(), m, 96, r*tile-20-2)
	w.Step(Input{Button: true})
	if w.Map.At(6, r) != level.Empty || len(w.Bullets) != 0 {
		t.Fatalf("tile %d bullets %d, want the block broken and no bullet", w.Map.At(6, r), len(w.Bullets))
	}
}

// gapCeiling returns a world with the player on the floor at x under a solid
// ceiling (row 8) that has a one-tile gap at x 96..111.
func gapCeiling(t *testing.T, x int) *World {
	t.Helper()
	w := roomWithRow(t, 8, level.Solid, x)
	w.Map.Set(6, 8, level.Empty)
	return w
}

func TestCornerCorrection(t *testing.T) {
	tests := []struct {
		name    string
		x       int
		in      Input
		through bool
	}{
		{"centred", 98, Input{}, true},
		{"1 px over the right corner", 101, Input{}, true},
		{"4 px over the right corner", 104, Input{}, true},
		{"5 px over the right corner", 105, Input{}, false},
		{"4 px over the left corner", 92, Input{}, true},
		{"5 px over the left corner", 91, Input{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := gapCeiling(t, tt.x)
			top := w.Player.Body.Y
			for i := 0; i < 20; i++ {
				w.Step(Input{Button: true})
				top = min(top, w.Player.Body.Y)
			}
			if through := top < 8*tile; through != tt.through {
				t.Fatalf("head top reached %d: through = %v, want %v", top, through, tt.through)
			}
		})
	}
}

func TestCornerCorrectionFollowsMovement(t *testing.T) {
	// 4 px over a corner, with a run speed so small that holding a direction
	// gives VX that sign without moving the player off the spot.
	tests := []struct {
		name    string
		x       int
		in      Input
		through bool
	}{
		{"right overhang moving right", 104, Input{Right: true}, false},
		{"right overhang moving left", 104, Input{Left: true}, true},
		{"left overhang moving left", 92, Input{Left: true}, false},
		{"left overhang moving right", 92, Input{Right: true}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tun := testTuning()
			tun.Player.RunSpeed = 0.001
			m := testRoom(t)
			for c := 1; c < 12; c++ {
				m.Set(c, 8, level.Solid)
			}
			m.Set(6, 8, level.Empty)
			w := NewWorld(tun, m, tt.x, floorY-tun.Player.Height)
			w.Step(Input{})
			top := w.Player.Body.Y
			for i := 0; i < 20; i++ {
				in := tt.in
				in.Button = true
				w.Step(in)
				top = min(top, w.Player.Body.Y)
			}
			if through := top < 8*tile; through != tt.through {
				t.Fatalf("head top reached %d: through = %v, want %v", top, through, tt.through)
			}
		})
	}
}

func TestBreakingLeavesTheChunkUnchanged(t *testing.T) {
	m := testRoom(t)
	for c := 1; c < 12; c++ {
		m.Set(c, 8, drill)
	}
	w := NewWorld(testTuning(), m, 96, floorY-20)
	w.Step(Input{})
	run(w, Input{Button: true}, 12)
	if w.Map.At(6, 8) != level.Empty {
		t.Fatal("block not broken")
	}
	if m.At(6, 8) != drill {
		t.Fatal("breaking a block in the world changed the chunk map")
	}
}

func TestUndefinedTileValue(t *testing.T) {
	m, err := level.ParseRows(tile, "#...........#", "#...........#", "#9..........#")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewWorldInChunk(testTuning(), m); err == nil || !strings.Contains(err.Error(), "value 9") {
		t.Fatalf("error = %v, want one about value 9", err)
	}
}

func TestSetTuningKeepsBlocksThatAreInUse(t *testing.T) {
	w := roomWithRow(t, 8, drill, 96)
	tun := testTuning()
	tun.Blocks = tun.Blocks[:1] // only solid
	tun.Player.RunSpeed = 90
	if err := w.SetTuning(tun); err == nil || !strings.Contains(err.Error(), "value 3") {
		t.Fatalf("error = %v, want one about value 3", err)
	}
	if got := w.Tuning(); len(got.Blocks) != len(testTuning().Blocks) || got.Player.RunSpeed != 90 {
		t.Fatalf("got %d blocks and run speed %v, want the old blocks and the new run speed",
			len(got.Blocks), got.Player.RunSpeed)
	}
	// A new block kind applies at once.
	tun = testTuning()
	tun.Blocks[2].OneWay = true // drill blocks become platforms
	if err := w.SetTuning(tun); err != nil {
		t.Fatal(err)
	}
	if w.Map.ShapeAt(6, 8) != level.ShapeOneWay {
		t.Fatal("changed block definition not applied to the map")
	}
}

// TestShippedTowerChunksCanBeClimbedThrough is a stopgap until the M2
// reachability validator: every row of every chunk the shipped tower draws
// from must have a cell the player can pass upward through, so no chunk
// seals the tower. Empty cells, one-way platforms, and blocks the head
// drills through all count.
func TestShippedTowerChunksCanBeClimbedThrough(t *testing.T) {
	tun, err := tuning.Load("../../data/tuning.json")
	if err != nil {
		t.Fatal(err)
	}
	chunks, err := level.LoadLDtk("../../assets/chunks/chunks.ldtk")
	if err != nil {
		t.Fatal(err)
	}
	bt := newBlockTable(tun.Blocks)
	for _, name := range tun.Tower.Pool {
		m, err := level.FindChunk(chunks, name)
		if err != nil {
			t.Fatal(err)
		}
		for r := 0; r < m.Rows; r++ {
			open := false
			for c := 1; c < m.Cols-1 && !open; c++ {
				v := m.At(c, r)
				open = v == level.Empty || bt[v].OneWay || bt[v].Drill
			}
			if !open {
				t.Errorf("chunk %s: row %d has no cell to climb through", name, r)
			}
		}
	}
}

func TestShippedChunksMatchShippedData(t *testing.T) {
	tun, err := tuning.Load("../../data/tuning.json")
	if err != nil {
		t.Fatal(err)
	}
	const path = "../../assets/chunks/chunks.ldtk"
	chunks, err := level.LoadLDtk(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range chunks {
		if err := checkLevel(tun, c.Map); err != nil {
			t.Errorf("chunk %s: %v", c.Name, err)
		}
	}
	names, err := level.EntityNames(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range names {
		if _, ok := enemyDef(tun, n); !ok {
			t.Errorf("LDtk entity %q has no enemy definition", n)
		}
	}
	values, err := level.CollisionValues(path)
	if err != nil {
		t.Fatal(err)
	}
	bt := newBlockTable(tun.Blocks)
	for _, v := range values {
		if v.Value < 1 || v.Value > 255 || bt[v.Value].Value == 0 {
			t.Errorf("LDtk value %d (%s) has no block definition", v.Value, v.Name)
			continue
		}
		if b := bt[v.Value]; b.Name != v.Name {
			t.Errorf("LDtk value %d is %q, but its block is %q", v.Value, v.Name, b.Name)
		}
	}
}

func TestRisingThroughOneWayIsNotLanding(t *testing.T) {
	// Fire the whole magazine from below a one-way platform. Crossing its top
	// edge on the way up must not count as landing: no refill and no end to
	// the fire stream. Whether a step ends with the feet exactly on the edge
	// depends on the start height, so try a whole tile of them.
	g := testTuning().Gun
	for d := 0; d < tile; d++ {
		m := level.NewTileMap(13, 200, tile)
		for c := 1; c < 12; c++ {
			m.Set(c, 100, oneWay) // top at y 1600
		}
		w := NewWorld(testTuning(), m, 96, 1600+48+d-20)
		shots := 0
		for i := 0; i < g.Magazine*6; i++ { // FireInterval is 6 steps
			fuel := w.Player.Fuel
			w.Step(Input{Button: true})
			if w.Player.Fuel < fuel {
				shots++
			}
			if w.Player.Fuel > fuel {
				t.Fatalf("start %d px lower: step %d refilled while rising (VY %v, feet %d)",
					d, i, w.Player.VY, w.Player.Body.Y+w.Player.Body.H)
			}
		}
		if shots != g.Magazine {
			t.Fatalf("start %d px lower: fired %d shots rising through the platform, want %d", d, shots, g.Magazine)
		}
	}
}

func TestOneWayAssistHittingADrillBlockBreaksIt(t *testing.T) {
	// Head touching a drill ceiling (row 8) while rising slowly through a
	// one-way platform (row 9): the main move rounds to 0 px and only the
	// assist reaches the ceiling, which must still drill it.
	m := testRoom(t)
	m.Set(6, 8, drill)
	m.Set(6, 9, oneWay)
	w := NewWorld(testTuning(), m, 98, 9*tile)
	w.Player.VY = -31 // -1 after one step of gravity
	w.Step(Input{})
	if w.Map.At(6, 8) != level.Empty {
		t.Fatal("drill block not broken by the assist move")
	}
}

func TestSetTuningKeepsBlocksThatWouldTrapThePlayer(t *testing.T) {
	m := testRoom(t)
	for c := 1; c < 12; c++ {
		m.Set(c, 9, oneWay)
	}
	w := NewWorld(testTuning(), m, 96, 9*tile-4) // overlaps the platform
	tun := testTuning()
	tun.Blocks[1].OneWay = false
	if err := w.SetTuning(tun); err == nil {
		t.Fatal("making the platform the player is inside solid gave no error")
	}
	if w.Map.ShapeAt(6, 9) != level.ShapeOneWay || !w.Tuning().Blocks[1].OneWay {
		t.Fatal("block definitions changed although they trap the player")
	}
}

func TestTuningBlocksAreNotShared(t *testing.T) {
	tun := testTuning()
	w := NewWorld(tun, testRoom(t), 96, 0)
	tun.Blocks[0].Value = 9
	got := w.Tuning()
	got.Blocks[1].Name = "changed"
	if b := w.Tuning().Blocks; b[0].Value != 1 || b[1].Name != "one_way" {
		t.Fatalf("world blocks changed through a caller's slice: %+v", b[:2])
	}
	tun = testTuning()
	if err := w.SetTuning(tun); err != nil {
		t.Fatal(err)
	}
	tun.Blocks[0].Value = 9
	if w.Tuning().Blocks[0].Value != 1 {
		t.Fatal("world blocks changed through the slice passed to SetTuning")
	}
}

func TestSpawnCheckUsesBlockShapes(t *testing.T) {
	// A one-way tile at the top centre does not block the spawn.
	m, err := level.ParseRows(tile, "#.....2.....#", "#...........#", "#############")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewWorldInChunk(testTuning(), m); err != nil {
		t.Fatalf("spawn over a one-way tile: %v", err)
	}
}

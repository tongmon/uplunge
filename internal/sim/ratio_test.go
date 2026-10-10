package sim

import (
	"testing"

	"github.com/tongmon/uplunge/internal/level"
)

func TestReachingTheTopClearsATowerRun(t *testing.T) {
	w := waterTower(t)
	w.Player.Body.Y, w.Player.VY = 3, -240
	w.Step(Input{})
	if !w.Cleared || w.ClearTick != w.Tick || w.Over {
		t.Fatalf("Cleared=%v ClearTick=%d Tick=%d Over=%v, want cleared on this tick", w.Cleared, w.ClearTick, w.Tick, w.Over)
	}
	p := w.Player
	run(w, Input{Button: true}, 10)
	if w.Player != p || w.ClearTick != w.Tick-10 {
		t.Fatal("the world kept changing after the clear")
	}
}

func TestChunkRunsHaveNoGoal(t *testing.T) {
	w, err := NewWorldInChunk(testTuning(), testRoom(t))
	if err != nil {
		t.Fatal(err)
	}
	w.Step(Input{}) // starts at the top
	if w.Cleared {
		t.Fatal("a single-chunk run cleared at its start")
	}
}

func TestEnemyGap(t *testing.T) {
	m := level.NewTileMap(13, 100, tile)
	m.Spawns = []level.Spawn{
		{Name: "Floater", X: 50, Y: 900},
		{Name: "Spiker", X: 50, Y: 700}, // not stompable: not counted
		{Name: "Floater", X: 50, Y: 500},
		{Name: "Floater", X: 50, Y: 600},
	}
	if got := EnemyGap(testTuning(), m); got != 200 {
		t.Fatalf("EnemyGap = %v, want (900-500)/2 = 200", got)
	}
	m.Spawns = m.Spawns[:2]
	if got := EnemyGap(testTuning(), m); got != 0 {
		t.Fatalf("EnemyGap with one Floater = %v, want 0", got)
	}
}

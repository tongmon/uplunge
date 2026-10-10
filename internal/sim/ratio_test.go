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
	if got, n := EnemyGap(testTuning(), m); got != 200 || n != 3 {
		t.Fatalf("EnemyGap = %v over %d, want (900-500)/2 = 200 over 3", got, n)
	}
	m.Spawns = m.Spawns[:2]
	if got, n := EnemyGap(testTuning(), m); got != 0 || n != 1 {
		t.Fatalf("EnemyGap with one Floater = %v over %d, want 0 over 1", got, n)
	}
}

func TestLosingTheLastHPAtTheTopIsNotAClear(t *testing.T) {
	w := waterTower(t)
	w.Player.HP = 1
	b := w.Player.Body
	w.Enemies = append(w.Enemies, Enemy{Def: testTuning().Enemies[1], Body: b, HP: 3})
	w.Enemies[0].Body.Y = 0
	w.Player.Body.Y, w.Player.VY = 0, 0
	w.Step(Input{})
	if !w.Over || w.Cleared {
		t.Fatalf("Over=%v Cleared=%v, want the run over, not cleared", w.Over, w.Cleared)
	}
}

func TestReach(t *testing.T) {
	tun := testTuning()
	ground, gdone := GroundReach(tun)
	stomp, sdone, ok := StompReach(tun)
	rise, _ := MagazineRise(tun)
	t.Logf("ground reach %d px, stomp reach %d px, magazine alone %d px", ground, stomp, rise)
	if !gdone || !sdone || !ok {
		t.Fatalf("measurements did not finish: ground %v stomp %v ok %v", gdone, sdone, ok)
	}
	// A jump or a bounce adds to the magazine, and a stomp's bounce is
	// stronger than a jump.
	if !(ground > rise && stomp > ground) {
		t.Fatalf("ground %d, stomp %d, magazine %d: want magazine < ground < stomp", ground, stomp, rise)
	}
	// The 2026-10-11 playtest of the lab (Floaters 12 px tall, jitter 16)
	// reached the first Floater from the floor at a spacing of 180 px,
	// barely, and never at 192. The lowest first Floater's top is spacing -
	// 16 + 6 px above the floor: 170 at 180 (reached), 182 at 192 (not).
	if ground < 170 || ground >= 182 {
		t.Fatalf("ground reach %d px, want 170 to under 182 to match the playtest", ground)
	}
	noStomp := testTuning()
	noStomp.Enemies = noStomp.Enemies[1:] // only the Spiker
	if _, _, ok := StompReach(noStomp); ok {
		t.Fatal("stomp reach measured without a stompable enemy")
	}
}

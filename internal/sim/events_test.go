package sim

import (
	"testing"

	"github.com/tongmon/uplunge/internal/level"
)

// stompWorld drops the player onto a still Floater and returns the world
// right after the stomp step.
func stompWorld(t *testing.T) *World {
	t.Helper()
	w := stillEnemyWorld(t, "Floater", 104, 1006, 98, 1000-20-40)
	for i := 0; i < Hz && !w.Events.Stomped; i++ {
		w.Step(Input{})
	}
	if !w.Events.Stomped {
		t.Fatal("no stomp")
	}
	return w
}

func TestStompFreezesTheWorld(t *testing.T) {
	w := stompWorld(t)
	p, tick := w.Player, w.Tick
	for i := 0; i < 3; i++ { // FreezeTime 0.05 s at 60 Hz
		w.Step(Input{Right: true})
		if !w.Events.Frozen || w.Player != p {
			t.Fatalf("frozen step %d: Frozen=%v, player changed=%v", i+1, w.Events.Frozen, w.Player != p)
		}
	}
	if w.Tick != tick+3 {
		t.Fatalf("Tick = %d, want the tick to keep counting through the freeze", w.Tick)
	}
	w.Step(Input{})
	if w.Events.Frozen || w.Player.Body.Y == p.Body.Y {
		t.Fatal("the world did not move again after the freeze")
	}
}

func TestDrillFreezesTheWorld(t *testing.T) {
	w := roomWithRow(t, 8, drill, 96)
	for i := 0; i < 20 && !w.Events.Drilled; i++ {
		w.Step(Input{Button: true})
	}
	if !w.Events.Drilled {
		t.Fatal("no drill break")
	}
	w.Step(Input{Button: true})
	if !w.Events.Frozen {
		t.Fatal("no freeze after the drill break")
	}
}

func TestLandingRefillDoesNotFreeze(t *testing.T) {
	w := NewWorld(testTuning(), testRoom(t), 96, 0)
	w.Step(Input{Button: true}) // spend a shot
	for i := 0; i < 2*Hz && !w.Events.Landed; i++ {
		w.Step(Input{})
	}
	if !w.Events.Landed || !w.Events.Refilled {
		t.Fatalf("landing events %+v, want Landed and Refilled", w.Events)
	}
	w.Step(Input{})
	if w.Events.Frozen {
		t.Fatal("a landing refill froze the world")
	}
}

func TestPressDuringAFreezeCounts(t *testing.T) {
	// Pressing during the freeze and holding on: the press registers on the
	// first step after it, so the shot is not lost.
	w := stompWorld(t)
	shots := 0
	for i := 0; i < 6; i++ {
		w.Step(Input{Button: true})
		if w.Events.Shot {
			shots++
		}
	}
	if shots != 1 {
		t.Fatalf("%d shots from a press held through the freeze, want 1", shots)
	}
}

func TestEvents(t *testing.T) {
	tun := testTuning()
	w := standingWorld(t)
	w.Step(Input{Button: true})
	if !w.Events.Jumped || w.Events.Shot {
		t.Fatalf("jump step events %+v, want Jumped only", w.Events)
	}
	run(w, Input{}, 4)
	w.Step(Input{Button: true})
	if !w.Events.Shot || w.Events.Refilled {
		t.Fatalf("shot step events %+v, want Shot", w.Events)
	}
	var landed Events
	for i := 0; i < 2*Hz; i++ {
		w.Step(Input{})
		if w.Events.Landed {
			landed = w.Events
			break
		}
	}
	if !landed.Landed || !landed.Refilled || landed.LandSpeed <= 0 || landed.LandSpeed > tun.Player.MaxFall {
		t.Fatalf("landing events %+v, want Landed and Refilled with a fall speed", landed)
	}
	w.Step(Input{})
	if w.Events.Landed || w.Events.Refilled {
		t.Fatalf("standing events %+v, want none", w.Events)
	}
}

func TestHurtAndCaughtEvents(t *testing.T) {
	m := level.NewTileMap(13, 200, tile)
	m.Spawns = []level.Spawn{{Name: "Spiker", X: 104, Y: 1010}}
	w := NewWorld(stillTuning(), m, 98, 995)
	w.Step(Input{})
	if !w.Events.Hurt {
		t.Fatalf("events %+v after touching a Spiker, want Hurt", w.Events)
	}
	ww := waterTower(t)
	ww.Player.invulnSteps = 5
	catchPlayer(t, ww)
	if !ww.Events.Caught || ww.Events.Hurt {
		t.Fatalf("events %+v after an immune catch, want Caught without Hurt", ww.Events)
	}
}

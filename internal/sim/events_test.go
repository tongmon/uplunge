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

func TestRefillCountsEvenWhenAShotSpentFuelTheSameStep(t *testing.T) {
	// A full magazine fires and the player is hurt in the same step: the
	// fuel ends where it started, but the hurt refilled it.
	m := level.NewTileMap(13, 200, tile)
	m.Spawns = []level.Spawn{{Name: "Spiker", X: 104, Y: 1012}}
	w := NewWorld(stillTuning(), m, 98, 1000)
	w.Step(Input{Button: true})
	if !w.Events.Shot || !w.Events.Hurt || !w.Events.Refilled {
		t.Fatalf("events %+v, want Shot, Hurt, and Refilled", w.Events)
	}
}

func TestRepressDuringAFreezeCounts(t *testing.T) {
	// Holding the button into the freeze, letting go and pressing again
	// during it, and holding on: the new press counts after the freeze.
	w := stompWorld(t)
	w.Player.prevButton = true // held before the stomp
	w.Step(Input{Button: true})
	w.Step(Input{})
	w.Step(Input{Button: true})
	if !w.Events.Frozen {
		t.Fatal("the freeze ended early")
	}
	w.Step(Input{Button: true})
	if !w.Events.Shot {
		t.Fatalf("events %+v after a press made during the freeze, want a shot", w.Events)
	}
}

func TestTapInsideAFreezeIsLost(t *testing.T) {
	w := stompWorld(t)
	w.Step(Input{Button: true})
	w.Step(Input{})
	w.Step(Input{})
	w.Step(Input{})
	if w.Events.Shot {
		t.Fatal("a tap made and released inside the freeze fired")
	}
}

func TestStartOnTheFloorIsNotALanding(t *testing.T) {
	w, err := NewWorldInTower(testTuning(), floorTower())
	if err != nil {
		t.Fatal(err)
	}
	if !w.Player.OnGround {
		t.Fatal("a player standing on the floor at the start is not on the ground")
	}
	w.Step(Input{})
	if w.Events.Landed {
		t.Fatal("standing still on the start floor reported a landing")
	}
}

func TestFreezeHoldsTheWholeWorld(t *testing.T) {
	m := level.NewTileMap(13, 200, tile)
	m.Spawns = []level.Spawn{
		{Name: "Floater", X: 104, Y: 1006},
		{Name: "Floater", X: 40, Y: 900}, // keeps flying, but not in a freeze
	}
	w := NewWorld(testTuning(), m, 98, 1000-20-40)
	w.Water = Water{On: true, Y: 1200}
	w.Player.Fuel = 1
	for i := 0; i < Hz && !w.Events.Stomped; i++ {
		w.Step(Input{})
	}
	if !w.Events.Stomped {
		t.Fatal("no stomp")
	}
	enemies := append([]Enemy(nil), w.Enemies...)
	bullets := append([]Bullet(nil), w.Bullets...)
	water, camera := w.Water, w.Camera
	for i := 0; i < 3; i++ {
		w.Step(Input{})
	}
	if len(w.Enemies) != len(enemies) || w.Enemies[0] != enemies[0] || len(w.Bullets) != len(bullets) ||
		w.Water != water || w.Camera != camera {
		t.Fatal("enemies, bullets, water, or camera moved during the freeze")
	}
}

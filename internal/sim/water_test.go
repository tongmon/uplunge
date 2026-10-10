package sim

import (
	"math"
	"testing"
)

// waterTower returns a tower world (floorTower) with the water on and the
// player standing on the bottom floor, so the camera rests at the map bottom.
func waterTower(t *testing.T) *World {
	t.Helper()
	w, err := NewWorldInTower(testTuning(), floorTower())
	if err != nil {
		t.Fatal(err)
	}
	if !w.Water.On {
		t.Fatal("tower world has no water")
	}
	return w
}

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestWaterStartsBelowTheMap(t *testing.T) {
	w := waterTower(t)
	if want := float64(200*tile) + testTuning().Water.StartBelow; w.Water.Y != want {
		t.Fatalf("water starts at %v, want %v", w.Water.Y, want)
	}
}

func TestChunkModeHasNoWater(t *testing.T) {
	m, err := NewWorldInChunk(testTuning(), testRoom(t))
	if err != nil {
		t.Fatal(err)
	}
	if m.Water.On {
		t.Fatal("chunk mode has water")
	}
}

func TestWaterSpeed(t *testing.T) {
	c := testTuning().Water
	tests := []struct {
		name string
		// below is how far under the baseline the water starts.
		below    float64
		wantRise float64
	}{
		{"at the baseline", 0, c.Speed * Dt},
		{"half the max lag behind", c.MaxLag / 2, 1.5 * c.Speed * Dt},
		{"at the max lag", c.MaxLag, c.MaxMult * c.Speed * Dt},
		{"further behind is pulled up to the max lag", c.MaxLag + 500, c.MaxMult * c.Speed * Dt},
		{"half the slow range above", -c.SlowRange / 2, 0.75 * c.Speed * Dt},
		{"well above", -c.SlowRange * 3, c.MinMult * c.Speed * Dt},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := waterTower(t)
			baseline := w.Camera.Bottom() - c.Baseline
			w.Water.Y = baseline + tt.below
			from := min(w.Water.Y, baseline+c.MaxLag)
			w.stepWater()
			if rise := from - w.Water.Y; !near(rise, tt.wantRise) {
				t.Fatalf("rose %v px in a step, want %v", rise, tt.wantRise)
			}
		})
	}
}

// catchPlayer raises the water to just over the standing player's feet and
// steps once, so the water catches the player.
func catchPlayer(t *testing.T, w *World) {
	t.Helper()
	b := w.Player.Body
	w.Water.Y = float64(b.Y+b.H) - 2
	w.Step(Input{})
	if w.Water.pauseSteps == 0 {
		t.Fatal("the water did not catch the player")
	}
}

func TestWaterSafetyNet(t *testing.T) {
	tun := testTuning()
	w := waterTower(t)
	w.Player.Fuel = 0
	catchPlayer(t, w)
	p := w.Player
	if p.HP != tun.Player.MaxHP-1 || p.Fuel != tun.Gun.Magazine || p.VY != -tun.Water.Bounce || !p.Invulnerable() {
		t.Fatalf("after the catch HP=%d Fuel=%d VY=%v invulnerable=%v, want 1 HP lost, refilled, launched, immune",
			p.HP, p.Fuel, p.VY, p.Invulnerable())
	}
	if p.coyoteSteps != 0 {
		t.Fatal("the launch left coyote time")
	}

	// The water eases down by Retreat over RetreatTime and stays put until
	// PauseTime after the catch, then rises again.
	from := w.Water.retreatFrom
	var ys []float64
	for i := 0; i < 40; i++ {
		w.Step(Input{})
		ys = append(ys, w.Water.Y)
	}
	retreat := 24 // RetreatTime 0.4 s
	pause := 30   // PauseTime 0.5 s from the catch
	if got := ys[retreat-1] - from; !near(got, tun.Water.Retreat) {
		t.Fatalf("water went down %v px by the end of the retreat, want %v", got, tun.Water.Retreat)
	}
	for i := retreat; i < pause-1; i++ {
		if ys[i] != ys[retreat-1] {
			t.Fatalf("water moved at step %d during the pause", i+1)
		}
	}
	if ys[pause-1] >= ys[pause-2] {
		t.Fatalf("water did not rise again once the pause was over (steps %d, %d: %v, %v)",
			pause-1, pause, ys[pause-2], ys[pause-1])
	}
}

func TestWaterRetreatEasesOut(t *testing.T) {
	w := waterTower(t)
	catchPlayer(t, w)
	from := w.Water.retreatFrom
	w.Step(Input{})
	first := w.Water.Y - from
	for i := 0; i < 11; i++ {
		w.Step(Input{})
	}
	half := w.Water.Y - from
	if !(first > half/12*1.5) || !(half > testTuning().Water.Retreat/2) {
		t.Fatalf("retreat went %v px in the first step and %v by halfway, want it fast first", first, half)
	}
}

func TestWaterHitWhileImmuneCostsNoHP(t *testing.T) {
	w := waterTower(t)
	w.Player.invulnSteps = 10
	catchPlayer(t, w)
	if w.Player.HP != testTuning().Player.MaxHP || w.Player.VY != -testTuning().Water.Bounce {
		t.Fatalf("HP %d VY %v, want no HP lost but still launched", w.Player.HP, w.Player.VY)
	}
}

func TestWaterEndsTheRunAtZeroHP(t *testing.T) {
	w := waterTower(t)
	w.Player.HP = 1
	catchPlayer(t, w)
	if !w.Over {
		t.Fatal("the run did not end when the water took the last HP")
	}
}

func TestWaterCatchesAStandingPlayer(t *testing.T) {
	// Left standing on the bottom floor, the player is caught soon after the
	// start: the water starts below the map and rises past the feet.
	w := waterTower(t)
	for i := 0; i < 3*Hz; i++ {
		w.Step(Input{})
		if w.Water.pauseSteps > 0 {
			t.Logf("caught after %.2f s", float64(i+1)/Hz)
			return
		}
	}
	t.Fatal("the water never reached the standing player")
}

func TestWaterDoesNotCatchAgainWhilePaused(t *testing.T) {
	w := waterTower(t)
	catchPlayer(t, w)
	// Hold the player under the surface for the rest of the pause: no
	// second launch, and the pause runs out instead of starting over.
	for i := 0; i < 25; i++ {
		pause := w.Water.pauseSteps
		w.Player.Body.Y = int(w.Water.Y) + 4
		w.Player.VY = 0
		w.Step(Input{})
		if w.Player.VY == -testTuning().Water.Bounce || w.Water.pauseSteps >= pause {
			t.Fatalf("step %d of the pause: caught again (VY %v, pause %d -> %d)", i, w.Player.VY, pause, w.Water.pauseSteps)
		}
	}
}

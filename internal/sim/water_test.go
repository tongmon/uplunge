package sim

import (
	"math"
	"testing"
)

// waterTower returns a tower world (floorTower) with the water on and
// already rising, and the player standing on the bottom floor, so the
// camera rests at the map bottom.
func waterTower(t *testing.T) *World {
	t.Helper()
	w := waitingTower(t)
	w.Water.waiting = false
	return w
}

// waitingTower is waterTower before the first jump, with the water waiting.
func waitingTower(t *testing.T) *World {
	t.Helper()
	w, err := NewWorldInTower(testTuning(), floorTower())
	if err != nil {
		t.Fatal(err)
	}
	if !w.Water.On || !w.Water.waiting {
		t.Fatal("tower world has no waiting water")
	}
	return w
}

func TestWaterWaitsForTheFirstJump(t *testing.T) {
	w := waitingTower(t)
	y := w.Water.Y
	run(w, Input{}, 3*Hz)
	run(w, Input{Right: true}, 10) // walking is not climbing
	run(w, Input{Left: true}, 10)
	if w.Water.Y != y || w.Player.HP != testTuning().Player.MaxHP {
		t.Fatalf("water moved from %v to %v before the first jump", y, w.Water.Y)
	}
	w.Step(Input{Button: true})
	if !(w.Water.Y < y) {
		t.Fatal("water did not start rising with the first jump")
	}
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
		{"half the max lag behind", c.MaxLag / 2, (1 + (c.MaxMult-1)/2) * c.Speed * Dt},
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
			w.stepWater(Input{})
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
	// Left standing on the bottom floor after a tap, the player is caught
	// soon: the water starts below the map and rises past the feet.
	w := waitingTower(t)
	w.Step(Input{Button: true})
	for i := 0; i < 5*Hz; i++ {
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
		w.Player.Body.Y = w.Water.Surface() + 4
		w.Player.VY = 0
		w.Step(Input{})
		if w.Player.VY == -testTuning().Water.Bounce || w.Water.pauseSteps >= pause {
			t.Fatalf("step %d of the pause: caught again (VY %v, pause %d -> %d)", i, w.Player.VY, pause, w.Water.pauseSteps)
		}
	}
}

func TestWaterCatchesAgainRightAfterThePause(t *testing.T) {
	w := waterTower(t)
	catchPlayer(t, w)
	// Keep the player under the surface: no catch for the rest of the
	// pause, then one on the step it ends.
	for i := 1; i <= 30; i++ {
		w.Player.Body.Y = w.Water.Surface() + 4
		w.Player.VY = 0
		w.Step(Input{})
		caught := w.Player.VY == -testTuning().Water.Bounce
		if want := i == 30; caught != want { // PauseTime 0.5 s from the catch
			t.Fatalf("step %d after the catch: caught = %v, want %v", i, caught, want)
		}
	}
}

func TestWaterCollidesAtTheDrawnSurface(t *testing.T) {
	w := waterTower(t)
	b := w.Player.Body
	feet := b.Y + b.H
	// A surface of feet - 0.4 rounds to feet: the water only touches the
	// feet's edge, so it must not catch yet.
	w.Water.Y = float64(feet) - 0.4
	w.touchWater()
	if w.Player.HP != testTuning().Player.MaxHP {
		t.Fatal("caught by water whose drawn surface only touches the feet")
	}
	w.Water.Y = float64(feet) - 0.6
	w.touchWater()
	if w.Player.HP != testTuning().Player.MaxHP-1 {
		t.Fatal("not caught by water drawn a pixel over the feet")
	}
}

func TestEnemyAndWaterInOneStepCostOneHP(t *testing.T) {
	// An enemy hit makes the player immune, so water reaching the player in
	// the same step only launches it.
	w := waterTower(t)
	b := w.Player.Body
	w.Enemies = append(w.Enemies, Enemy{Def: testTuning().Enemies[1], Body: b, HP: 3})
	w.Water.Y = float64(b.Y+b.H) - 2
	w.Step(Input{})
	if w.Player.HP != testTuning().Player.MaxHP-1 || w.Player.VY != -testTuning().Water.Bounce {
		t.Fatalf("HP %d VY %v, want one HP lost and the water's launch", w.Player.HP, w.Player.VY)
	}
}

func TestRetreatKeepsTheValuesFromTheCatch(t *testing.T) {
	w := waterTower(t)
	catchPlayer(t, w)
	from := w.Water.retreatFrom
	tun := testTuning()
	tun.Water.Retreat = 10
	tun.Water.RetreatTime = 2
	tun.Water.PauseTime = 3
	if err := w.SetTuning(tun); err != nil {
		t.Fatal(err)
	}
	run(w, Input{}, 24)
	if got := w.Water.Y - from; !near(got, testTuning().Water.Retreat) {
		t.Fatalf("retreated %v px after a reload mid-retreat, want the %v px set at the catch",
			got, testTuning().Water.Retreat)
	}
}

func TestRetreatGoesPastTheMaxLag(t *testing.T) {
	// The retreat is longer than the max lag; the water still goes the
	// whole way down, and only after the pause is it pulled back up to the
	// max lag, which is below the view.
	c := testTuning().Water
	if c.Retreat <= c.MaxLag {
		t.Skip("the retreat fits inside the max lag")
	}
	w := waterTower(t)
	catchPlayer(t, w)
	from := w.Water.retreatFrom
	run(w, Input{}, 29) // to the last step of the pause
	if got := w.Water.Y - from; !near(got, c.Retreat) {
		t.Fatalf("water went down %v px during the pause, want the whole %v", got, c.Retreat)
	}
	w.Step(Input{})
	baseline := w.Camera.Bottom() - c.Baseline
	if w.Water.Y > baseline+c.MaxLag || float64(w.Water.Surface()) < w.Camera.Bottom() {
		t.Fatalf("water at %v after the pause, want it pulled up to the max lag %v, still below the view",
			w.Water.Y, baseline+c.MaxLag)
	}
}

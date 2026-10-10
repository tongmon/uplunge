package sim

import (
	"testing"

	"github.com/tongmon/uplunge/internal/level"
)

// tallWorld returns a world with the player falling from rest at y 1600 of a
// 13x200-tile empty map, so firing never meets the floor or the top.
func tallWorld() *World {
	return NewWorld(testTuning(), level.NewTileMap(13, 200, tile), 96, 1600)
}

func TestShotThrustsAndSpendsFuel(t *testing.T) {
	g := testTuning().Gun
	w := tallWorld()
	w.Step(Input{})
	w.Step(Input{Button: true})
	if w.Player.VY != -g.Thrust {
		t.Fatalf("VY = %v after a shot, want %v", w.Player.VY, -g.Thrust)
	}
	if w.Player.Fuel != g.Magazine-1 {
		t.Fatalf("Fuel = %d, want %d", w.Player.Fuel, g.Magazine-1)
	}
	if len(w.Bullets) != 1 {
		t.Fatalf("%d bullets, want 1", len(w.Bullets))
	}
	b, p := w.Bullets[0].Body, w.Player.Body
	if b.Y != p.Y+p.H || b.X+b.W/2 != p.X+p.W/2 {
		t.Fatalf("bullet at (%d, %d), want centred under the player's feet at y %d", b.X, b.Y, p.Y+p.H)
	}
}

func TestHeldButtonFiresAtFireInterval(t *testing.T) {
	g := testTuning().Gun
	w := tallWorld()
	w.Step(Input{})
	var shots []int
	for i := 0; i < 2*Hz; i++ {
		fuel := w.Player.Fuel
		w.Step(Input{Button: true})
		if w.Player.Fuel < fuel {
			shots = append(shots, i)
		}
	}
	if len(shots) != g.Magazine {
		t.Fatalf("fired %d shots holding the button for 2 s, want the magazine of %d", len(shots), g.Magazine)
	}
	for i := 1; i < len(shots); i++ {
		if gap := shots[i] - shots[i-1]; gap != 6 { // FireInterval 0.1 s at 60 Hz
			t.Fatalf("shots %d and %d are %d steps apart, want 6", i-1, i, gap)
		}
	}
}

func TestEmptyMagazineDoesNothing(t *testing.T) {
	w := tallWorld()
	w.Step(Input{})
	w.Player.Fuel = 0
	w.Step(Input{Button: true})
	if w.Player.VY < 0 || len(w.Bullets) != 0 {
		t.Fatalf("VY = %v with %d bullets after firing on empty, want no thrust and no bullet",
			w.Player.VY, len(w.Bullets))
	}
}

func TestJumpPressDoesNotFire(t *testing.T) {
	g := testTuning().Gun
	w := standingWorld(t)
	// Holding the button from the jump through the whole arc never fires.
	for i := 0; i < Hz/2; i++ {
		w.Step(Input{Button: true})
	}
	if w.Player.Fuel != g.Magazine || len(w.Bullets) != 0 {
		t.Fatalf("Fuel = %d with %d bullets after holding the jump, want no shot", w.Player.Fuel, len(w.Bullets))
	}
	// Releasing and pressing again in the air fires.
	w = standingWorld(t)
	run(w, Input{Button: true}, 10)
	w.Step(Input{})
	w.Step(Input{Button: true})
	if w.Player.Fuel != g.Magazine-1 || w.Player.VY != -g.Thrust {
		t.Fatalf("Fuel = %d VY = %v after pressing again in the air, want one shot", w.Player.Fuel, w.Player.VY)
	}
}

func TestLandingRefills(t *testing.T) {
	g := testTuning().Gun
	w := NewWorld(testTuning(), testRoom(t), 96, 0)
	w.Step(Input{Button: true})
	run(w, Input{}, 2*Hz)
	if !w.Player.OnGround {
		t.Fatal("player did not land")
	}
	w.Step(Input{})
	if w.Player.Fuel != g.Magazine {
		t.Fatalf("Fuel = %d after landing, want %d", w.Player.Fuel, g.Magazine)
	}
}

func TestLandingEndsTheFireStream(t *testing.T) {
	g := testTuning().Gun
	m := ledgeWorld(t).Map
	// Start 40 px above the ledge, fire the whole magazine, keep holding
	// through the landing, then walk off the ledge still holding.
	w := NewWorld(testTuning(), m, 6*tile-12-2, 9*tile-20-40)
	run(w, Input{Button: true}, 3*Hz)
	if !w.Player.OnGround || w.Player.Fuel != g.Magazine {
		t.Fatalf("OnGround = %v Fuel = %d, want landed on the ledge and refilled", w.Player.OnGround, w.Player.Fuel)
	}
	for i := 0; w.Player.OnGround; i++ {
		if i > Hz {
			t.Fatal("player did not walk off the ledge within 1 s")
		}
		w.Step(Input{Button: true, Right: true})
	}
	run(w, Input{Button: true, Right: true}, 10)
	if w.Player.OnGround || w.Player.Fuel != g.Magazine {
		t.Fatalf("OnGround = %v Fuel = %d after walking off holding, want in the air with no shot",
			w.Player.OnGround, w.Player.Fuel)
	}
}

func TestFiringPressIsNotBuffered(t *testing.T) {
	land := landingStep(t)
	w := NewWorld(testTuning(), testRoom(t), 96, 0)
	run(w, Input{}, land-2)
	w.Step(Input{Button: true}) // fires
	if w.Player.Fuel != testTuning().Gun.Magazine-1 {
		t.Fatal("the press did not fire")
	}
	// The shot lifts the player a little; once back on the floor the player
	// must stay there instead of jumping from the same press.
	for i := 0; !w.Player.OnGround; i++ {
		if i > Hz {
			t.Fatal("player did not land within 1 s of the shot")
		}
		w.Step(Input{})
	}
	run(w, Input{}, 3)
	if !w.Player.OnGround || w.Player.VY != 0 {
		t.Fatalf("VY = %v OnGround = %v after landing, want the fired press used up", w.Player.VY, w.Player.OnGround)
	}
}

func TestEmptyPressIsBuffered(t *testing.T) {
	p := testTuning().Player
	land := landingStep(t)
	w := NewWorld(testTuning(), testRoom(t), 96, 0)
	run(w, Input{}, land-2)
	w.Player.Fuel = 0
	w.Step(Input{Button: true}) // nothing to fire: remembered as a jump
	run(w, Input{}, 2)
	if w.Player.VY != -p.JumpSpeed {
		t.Fatalf("VY = %v after landing, want the buffered jump", w.Player.VY)
	}
}

func TestBulletFliesForBulletLife(t *testing.T) {
	g := testTuning().Gun
	w := tallWorld()
	w.Step(Input{})
	w.Step(Input{Button: true})
	y0 := w.Bullets[0].Body.Y
	life := 12 // BulletLife 0.2 s at 60 Hz
	run(w, Input{}, life-1)
	if len(w.Bullets) != 1 {
		t.Fatalf("bullet gone after %d steps, want it to last %d", life-1, life)
	}
	if got, want := w.Bullets[0].Body.Y-y0, int(g.BulletSpeed*Dt)*(life-1); got != want {
		t.Fatalf("bullet fell %d px in %d steps, want %d", got, life-1, want)
	}
	w.Step(Input{})
	if len(w.Bullets) != 0 {
		t.Fatalf("bullet still flying after %d steps", life)
	}
}

func TestBulletStopsAtSolid(t *testing.T) {
	w := NewWorld(testTuning(), testRoom(t), 96, floorY-20-24)
	w.Step(Input{Button: true})
	if len(w.Bullets) != 1 {
		t.Fatal("no bullet fired")
	}
	for i := 0; i < 5 && len(w.Bullets) > 0; i++ {
		w.Step(Input{})
		if len(w.Bullets) > 0 {
			if b := w.Bullets[0].Body; b.Y+b.H > floorY {
				t.Fatalf("bullet bottom at %d, inside the floor at %d", b.Y+b.H, floorY)
			}
		}
	}
	if len(w.Bullets) != 0 {
		t.Fatalf("bullet at y %d still flying, want it stopped by the floor at %d", w.Bullets[0].Body.Y, floorY)
	}
}

func TestSetTuningClampsFuel(t *testing.T) {
	w := tallWorld()
	tun := testTuning()
	tun.Gun.Magazine = 3
	if err := w.SetTuning(tun); err != nil {
		t.Fatal(err)
	}
	if w.Player.Fuel != 3 {
		t.Fatalf("Fuel = %d after shrinking the magazine to 3", w.Player.Fuel)
	}
}

// magazineRise holds the button from rest in mid-air until the magazine is
// empty and the player peaks, and returns how many pixels the player rose.
func magazineRise(t *testing.T, apexGravMult float64) int {
	t.Helper()
	tun := testTuning()
	tun.Player.ApexGravMult = apexGravMult
	w := NewWorld(tun, level.NewTileMap(13, 200, tile), 96, 1600)
	y0, top := w.Player.Body.Y, w.Player.Body.Y
	for i := 0; i < 3*Hz; i++ {
		w.Step(Input{Button: true})
		top = min(top, w.Player.Body.Y)
	}
	if w.Player.Fuel != 0 {
		t.Fatalf("magazine not empty after 3 s: Fuel = %d", w.Player.Fuel)
	}
	return y0 - top
}

// TestMagazineRise measures the numerator of the core tuning ratio
// (docs/design.md section 4): how high one magazine lifts the player.
func TestMagazineRise(t *testing.T) {
	with, without := magazineRise(t, testTuning().Player.ApexGravMult), magazineRise(t, 1)
	t.Logf("one magazine from rest: %d px (%.2f tiles); without apex half gravity: %d px",
		with, float64(with)/tile, without)
	if with < 6*tile || with > 10*tile {
		t.Fatalf("one magazine rose %d px, want 6 to 10 tiles", with)
	}
	if with < without {
		t.Fatalf("apex half gravity lowered the rise: %d < %d px", with, without)
	}
}

func TestRefillOnTheLandingStep(t *testing.T) {
	w := NewWorld(testTuning(), testRoom(t), 96, 0)
	w.Step(Input{Button: true})
	for i := 0; !w.Player.OnGround; i++ {
		if i > 2*Hz {
			t.Fatal("player did not land within 2 s")
		}
		w.Step(Input{})
	}
	if w.Player.Fuel != testTuning().Gun.Magazine {
		t.Fatalf("Fuel = %d on the landing step, want it refilled", w.Player.Fuel)
	}
}

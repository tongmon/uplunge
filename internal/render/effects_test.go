package render

import (
	"math"
	"reflect"
	"testing"

	"github.com/tongmon/uplunge/internal/level"
	"github.com/tongmon/uplunge/internal/sim"
	"github.com/tongmon/uplunge/internal/tuning"
)

// world returns a world with the replay testdata tuning, whose Events the
// tests set by hand.
func world(t *testing.T) *sim.World {
	t.Helper()
	tun, err := tuning.Load("../replay/testdata/tuning.json")
	if err != nil {
		t.Fatal(err)
	}
	return sim.NewWorld(tun, level.NewTileMap(13, 30, 16), 96, 100)
}

// stepWith steps fx once with the given events.
func stepWith(fx *Effects, w *sim.World, ev sim.Events) {
	w.Events = ev
	fx.Step(w)
}

func TestShake(t *testing.T) {
	w := world(t)
	f := w.Tuning().Feel
	fx := NewEffects()
	stepWith(&fx, w, sim.Events{Shot: true})
	// The first offset pushes the view up (against the downward shot) by
	// ShakeScale px per second left, rounded up: 0.2 s * 20 = 4 px.
	if got := fx.ShakeY(); got != -4 {
		t.Fatalf("first shake offset %d, want -4", got)
	}
	var offsets []int
	steps := 0
	for i := 0; i < sim.Hz && fx.ShakeY() != 0; i++ {
		offsets = append(offsets, fx.ShakeY())
		stepWith(&fx, w, sim.Events{})
		steps++
	}
	if want := int(math.Round(f.ShakeTime * sim.Hz)); steps < want-1 || steps > want+1 {
		t.Fatalf("shook for %d steps, want about ShakeTime (%d)", steps, want)
	}
	flips, prev := 0, offsets[0]
	for _, o := range offsets[1:] {
		if o != prev && (o < 0) != (prev < 0) {
			flips++
		}
		if abs(o) > abs(offsets[0]) {
			t.Fatalf("shake grew from %d to %d", offsets[0], o)
		}
		prev = o
	}
	if flips < 3 {
		t.Fatalf("offsets %v flip side %d times, want it to alternate", offsets, flips)
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func TestSquashAndStretch(t *testing.T) {
	w := world(t)
	f := w.Tuning().Feel
	fx := NewEffects()
	stepWith(&fx, w, sim.Events{Jumped: true})
	if x, y := fx.Scale(); x != f.JumpX || y != f.JumpY {
		t.Fatalf("jump scale %v, %v, want %v, %v", x, y, f.JumpX, f.JumpY)
	}
	// Eases back to 1 at Recover per second: 0.4 of stretch takes 0.23 s.
	for i := 0; i < sim.Hz/2; i++ {
		stepWith(&fx, w, sim.Events{})
	}
	if x, y := fx.Scale(); x != 1 || y != 1 {
		t.Fatalf("scale %v, %v half a second after the jump, want back to 1", x, y)
	}
	stepWith(&fx, w, sim.Events{Landed: true, LandSpeed: f.LandSpeed * 2})
	if x, y := fx.Scale(); x != f.LandX || y != f.LandY {
		t.Fatalf("hard landing scale %v, %v, want the full %v, %v", x, y, f.LandX, f.LandY)
	}
	fx = NewEffects()
	stepWith(&fx, w, sim.Events{Landed: true, LandSpeed: f.LandSpeed / 2})
	if x, y := fx.Scale(); math.Abs(x-(1+f.LandX)/2) > 1e-9 || math.Abs(y-(1+f.LandY)/2) > 1e-9 {
		t.Fatalf("landing at half speed scale %v, %v, want halfway", x, y)
	}
}

func TestRefillFlash(t *testing.T) {
	w := world(t)
	fx := NewEffects()
	stepWith(&fx, w, sim.Events{Refilled: true})
	n := 1
	for fx.Flashing() && n < sim.Hz {
		stepWith(&fx, w, sim.Events{})
		if fx.Flashing() {
			n++
		}
	}
	if n != 7 { // FlashTime 0.12 s at 60 Hz
		t.Fatalf("flashed for %d steps, want 7", n)
	}
}

func TestFrozenStepsChangeNothing(t *testing.T) {
	w := world(t)
	fx := NewEffects()
	stepWith(&fx, w, sim.Events{Shot: true, Jumped: true, Refilled: true})
	fx.burst(sim.BrokenCell{Col: 3, Row: 3, Tile: 3}, 16, w.Tuning().Feel)
	before := fx
	before.Debris = append([]Piece(nil), fx.Debris...)
	for i := 0; i < 3; i++ {
		stepWith(&fx, w, sim.Events{Frozen: true})
	}
	if !reflect.DeepEqual(fx, before) {
		t.Fatal("effects changed during a freeze")
	}
}

func TestVisibleRowsFollowTheShake(t *testing.T) {
	// The camera at y 160 shows rows 10..32; shaken 4 px down, the picture
	// moves down and the 4 px above it come from row 9.
	for _, tt := range []struct {
		camY, shake, r0, r1 int
	}{
		{160, 0, 10, 32},
		{160, 4, 9, 32},
		{160, -4, 10, 32},
		{168, -4, 10, 33},
	} {
		r0, r1 := visibleRows(tt.camY, tt.shake, 16, 100)
		if r0 != tt.r0 || r1 != tt.r1 {
			t.Errorf("camY %d shake %d: rows %d..%d, want %d..%d", tt.camY, tt.shake, r0, r1, tt.r0, tt.r1)
		}
	}
}

func TestStompAndCatchStretch(t *testing.T) {
	w := world(t)
	f := w.Tuning().Feel
	for _, ev := range []sim.Events{{Stomped: true}, {Caught: true}} {
		fx := NewEffects()
		stepWith(&fx, w, ev)
		if x, y := fx.Scale(); x != f.JumpX || y != f.JumpY {
			t.Fatalf("events %+v: scale %v, %v, want the jump's %v, %v", ev, x, y, f.JumpX, f.JumpY)
		}
	}
}

func TestStompShake(t *testing.T) {
	w := world(t)
	fx := NewEffects()
	stepWith(&fx, w, sim.Events{Stomped: true})
	// 0.16 s * 12 px/s = 1.92, rounded up: 2 px, as Downwell's 2 px stomp
	// shake.
	if got := fx.ShakeY(); got != -2 {
		t.Fatalf("stomp shake offset %d, want -2", got)
	}
	steps := 0
	for fx.ShakeY() != 0 && steps < sim.Hz {
		stepWith(&fx, w, sim.Events{})
		steps++
	}
	if steps < 9 || steps > 11 { // StompShakeTime 0.16 s at 60 Hz
		t.Fatalf("stomp shook for %d steps, want about 10", steps)
	}
}

func TestShakesDoNotMix(t *testing.T) {
	w := world(t)
	// A stomp during a shot's shake: the stronger of the two goes on whole,
	// never the shot's time with the stomp's scale.
	fx := NewEffects()
	stepWith(&fx, w, sim.Events{Shot: true})
	stepWith(&fx, w, sim.Events{Stomped: true})
	if got := abs(fx.ShakeY()); got != 4 && got != 3 {
		t.Fatalf("shake %d px after a stomp during a shot's shake, want the shot's", got)
	}
	// Both in one step: the shot's stronger shake.
	fx = NewEffects()
	stepWith(&fx, w, sim.Events{Shot: true, Stomped: true})
	if got := abs(fx.ShakeY()); got != 4 {
		t.Fatalf("shake %d px for a shot and a stomp at once, want the shot's 4", got)
	}
	// A shot late in a stomp's shake takes over.
	fx = NewEffects()
	stepWith(&fx, w, sim.Events{Stomped: true})
	for i := 0; i < 8; i++ {
		stepWith(&fx, w, sim.Events{})
	}
	stepWith(&fx, w, sim.Events{Shot: true})
	if got := abs(fx.ShakeY()); got != 4 {
		t.Fatalf("shake %d px for a shot late in a stomp's shake, want the shot's 4", got)
	}
}

func TestBounceBeatsALandingInTheSameStep(t *testing.T) {
	w := world(t)
	f := w.Tuning().Feel
	for _, ev := range []sim.Events{
		{Landed: true, LandSpeed: f.LandSpeed, Stomped: true},
		{Landed: true, LandSpeed: f.LandSpeed, Caught: true},
	} {
		fx := NewEffects()
		stepWith(&fx, w, ev)
		if x, y := fx.Scale(); x != f.JumpX || y != f.JumpY {
			t.Fatalf("events %+v: scale %v, %v, want the bounce's stretch", ev, x, y)
		}
	}
}

func TestDrillShakeAndStretch(t *testing.T) {
	w := world(t)
	f := w.Tuning().Feel
	fx := NewEffects()
	stepWith(&fx, w, sim.Events{Drilled: true})
	// 0.2 s * 15 px/s = 3 px.
	if got := fx.ShakeY(); got != -3 {
		t.Fatalf("drill shake offset %d, want -3", got)
	}
	if x, y := fx.Scale(); x != f.JumpX || y != f.JumpY {
		t.Fatalf("drill scale %v, %v, want the jump's stretch", x, y)
	}
}

func TestDebris(t *testing.T) {
	w := world(t)
	f := w.Tuning().Feel
	fx := NewEffects()
	ev := sim.Events{Drilled: true, NBroken: 2}
	ev.Broken[0] = sim.BrokenCell{Col: 3, Row: 5, Tile: 3}
	ev.Broken[1] = sim.BrokenCell{Col: 7, Row: 9, Tile: 4, ByBullet: true}
	stepWith(&fx, w, ev)
	if len(fx.Debris) != 8 {
		t.Fatalf("%d pieces from two broken tiles, want 8", len(fx.Debris))
	}
	// Pieces start inside their tile, apart from each other.
	for i, p := range fx.Debris[:4] {
		if p.X < 48 || p.X+pieceSize > 64 || p.Y < 80 || p.Y+pieceSize > 96 || p.Tile != 3 {
			t.Fatalf("piece %d %+v starts outside its tile (3, 5)", i, p)
		}
	}
	// Drilled pieces fly up, bulleted ones down; both fly outward.
	for i, p := range fx.Debris {
		up := p.VY < 0
		if up == (i >= 4) {
			t.Fatalf("piece %d VY %v: want drill debris up and bullet debris down", i, p.VY)
		}
		if (i%2 == 0) != (p.VX < 0) {
			t.Fatalf("piece %d VX %v: want the left pieces left and the right ones right", i, p.VX)
		}
	}
	// They fall, and are gone after DebrisLife.
	y := fx.Debris[0].Y
	life := int(math.Round(f.DebrisLife * sim.Hz))
	for i := 0; i < life-1; i++ {
		stepWith(&fx, w, sim.Events{})
	}
	if len(fx.Debris) == 0 || fx.Debris[0].Y <= y {
		t.Fatal("debris gone early or never fell")
	}
	stepWith(&fx, w, sim.Events{})
	if len(fx.Debris) != 0 {
		t.Fatalf("%d pieces left after DebrisLife", len(fx.Debris))
	}
}

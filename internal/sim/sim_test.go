package sim

import (
	"testing"

	"github.com/tongmon/uplunge/internal/level"
	"github.com/tongmon/uplunge/internal/tuning"
)

// testTuning mirrors data/tuning.json so these tests pin the behavior of the
// current hypotheses rather than whatever the file says later.
func testTuning() tuning.Tuning {
	return tuning.Tuning{Player: tuning.Player{
		Width: 12, Height: 20,
		Gravity: 1800, MaxFall: 320,
		JumpSpeed: 210, JumpHoldTime: 0.2,
		RunSpeed: 180, RunAccel: 2000, AirAccelMult: 0.65,
	}}
}

const tile = 16

// testRoom is 13x12 tiles: walls on both sides and a floor whose top is at
// y = 11*16 = 176. The inner space spans x 16..191.
func testRoom(t *testing.T) *level.TileMap {
	t.Helper()
	m, err := level.ParseRows(tile,
		"#...........#",
		"#...........#",
		"#...........#",
		"#...........#",
		"#...........#",
		"#...........#",
		"#...........#",
		"#...........#",
		"#...........#",
		"#...........#",
		"#...........#",
		"#############",
	)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

const floorY = 11 * tile

// standingWorld returns a world with the player standing on the floor.
func standingWorld(t *testing.T) *World {
	t.Helper()
	tun := testTuning()
	w := NewWorld(tun, testRoom(t), 96, floorY-tun.Player.Height)
	w.Step(Input{})
	if !w.Player.OnGround {
		t.Fatal("player does not start on the ground")
	}
	return w
}

func run(w *World, in Input, steps int) {
	for i := 0; i < steps; i++ {
		w.Step(in)
	}
}

func TestStepAdvancesOneTick(t *testing.T) {
	w := NewWorld(testTuning(), testRoom(t), 96, 0)
	run(w, Input{}, 3*Hz)
	if got, want := w.Tick, uint64(3*Hz); got != want {
		t.Fatalf("Tick = %d, want %d", got, want)
	}
}

func TestDtMatchesHz(t *testing.T) {
	if got := Dt * Hz; got != 1 {
		t.Fatalf("Dt * Hz = %v, want 1", got)
	}
}

func TestFallsAndLands(t *testing.T) {
	tun := testTuning()
	w := NewWorld(tun, testRoom(t), 96, 0)
	run(w, Input{}, Hz)
	p := w.Player
	if !p.OnGround || p.VY != 0 || p.Body.Y != floorY-tun.Player.Height {
		t.Fatalf("after 1 s: Y=%d VY=%v OnGround=%v, want Y=%d VY=0 on ground",
			p.Body.Y, p.VY, p.OnGround, floorY-tun.Player.Height)
	}
}

func TestFallSpeedIsCapped(t *testing.T) {
	tun := testTuning()
	tall := level.NewTileMap(13, 200, tile)
	w := NewWorld(tun, tall, 96, 0)
	run(w, Input{}, Hz)
	if w.Player.VY != tun.Player.MaxFall {
		t.Fatalf("VY = %v after 1 s of falling, want %v", w.Player.VY, tun.Player.MaxFall)
	}
}

func TestRunReachesRunSpeed(t *testing.T) {
	w := standingWorld(t)
	x0 := w.Player.Body.X
	run(w, Input{Right: true}, Hz/4)
	if w.Player.VX != testTuning().Player.RunSpeed {
		t.Fatalf("VX = %v, want %v", w.Player.VX, testTuning().Player.RunSpeed)
	}
	if w.Player.Body.X <= x0 {
		t.Fatalf("X = %d, want more than %d", w.Player.Body.X, x0)
	}
}

func TestAirAccelIsReduced(t *testing.T) {
	ground := standingWorld(t)
	ground.Step(Input{Right: true})

	air := NewWorld(testTuning(), testRoom(t), 96, 0)
	air.Step(Input{Right: true})

	p := testTuning().Player
	if got, want := ground.Player.VX, p.RunAccel*Dt; got != want {
		t.Errorf("ground VX after one step = %v, want %v", got, want)
	}
	if got, want := air.Player.VX, p.RunAccel*p.AirAccelMult*Dt; got != want {
		t.Errorf("air VX after one step = %v, want %v", got, want)
	}
}

func TestRunStopsAtWall(t *testing.T) {
	w := standingWorld(t)
	run(w, Input{Right: true}, 2*Hz)
	wantX := 12*tile - testTuning().Player.Width
	if w.Player.Body.X != wantX || w.Player.VX != 0 {
		t.Fatalf("X=%d VX=%v, want X=%d VX=0", w.Player.Body.X, w.Player.VX, wantX)
	}
}

// jumpHeight presses the button on a standing player, keeps it down for
// holdSteps, and returns how many pixels the player rose at the peak.
func jumpHeight(t *testing.T, holdSteps int) int {
	w := standingWorld(t)
	y0, top := w.Player.Body.Y, w.Player.Body.Y
	for i := 0; i < 2*Hz; i++ {
		w.Step(Input{Button: i < holdSteps})
		top = min(top, w.Player.Body.Y)
	}
	if !w.Player.OnGround {
		t.Fatal("player did not land within 2 s")
	}
	return y0 - top
}

func TestHeldJumpRisesAboutThreeAndAHalfTiles(t *testing.T) {
	h := jumpHeight(t, Hz)
	t.Logf("held jump height: %d px (%.2f tiles)", h, float64(h)/tile)
	if h < 3*tile || h > 4*tile {
		t.Fatalf("held jump rose %d px, want 3 to 4 tiles (%d..%d)", h, 3*tile, 4*tile)
	}
}

func TestJumpHoldLastsJumpHoldTime(t *testing.T) {
	p := testTuning().Player
	w := standingWorld(t)
	held := 0
	for i := 0; i < Hz; i++ {
		w.Step(Input{Button: true})
		if w.Player.VY == -p.JumpSpeed {
			held++
		}
	}
	// The launch step counts as the first held step.
	if want := int(p.JumpHoldTime * Hz); held != want {
		t.Fatalf("VY stayed at -JumpSpeed for %d steps, want %d", held, want)
	}
}

func TestTapJumpIsLower(t *testing.T) {
	tap, held := jumpHeight(t, 1), jumpHeight(t, Hz)
	t.Logf("tap jump height: %d px", tap)
	if tap <= 0 || tap >= held {
		t.Fatalf("tap jump rose %d px, want between 0 and the held %d px", tap, held)
	}
}

func TestNoJumpInAir(t *testing.T) {
	w := NewWorld(testTuning(), testRoom(t), 96, 0)
	w.Step(Input{})
	w.Step(Input{Button: true})
	if w.Player.VY < 0 {
		t.Fatalf("VY = %v after pressing in the air, want no jump", w.Player.VY)
	}
}

func TestHoldingButtonDoesNotRejump(t *testing.T) {
	w := standingWorld(t)
	run(w, Input{Button: true}, 2*Hz)
	if !w.Player.OnGround {
		t.Fatal("player is not on the ground after holding the button for 2 s")
	}
	run(w, Input{Button: true}, 10)
	if !w.Player.OnGround || w.Player.VY != 0 {
		t.Fatalf("player jumped again without a new press: VY=%v", w.Player.VY)
	}
}

func TestHeadBumpEndsJump(t *testing.T) {
	tun := testTuning()
	m := testRoom(t)
	// A ceiling whose underside is two tiles above the floor.
	for c := 1; c < 12; c++ {
		m.Set(c, 8, level.Solid)
	}
	w := NewWorld(tun, m, 96, floorY-tun.Player.Height)
	w.Step(Input{})
	top := w.Player.Body.Y
	// Hold for less than JumpHoldTime: without the cancel the head would stay
	// pinned to the ceiling for the whole hold.
	for i := 0; i < int(tun.Player.JumpHoldTime*Hz)-1; i++ {
		w.Step(Input{Button: true})
		top = min(top, w.Player.Body.Y)
	}
	if top != 9*tile {
		t.Fatalf("top Y = %d, want head against the ceiling at %d", top, 9*tile)
	}
	if w.Player.Body.Y == top {
		t.Fatal("player still pinned to the ceiling, want the hold cancelled")
	}
}

func TestNewWorldInChunk(t *testing.T) {
	tests := []struct {
		name         string
		rows         []string
		wantX, wantY int
		wantErr      bool
	}{
		{"open top", []string{"#...........#", "#...........#", "#############"}, 98, 0, false},
		{"solid at top centre", []string{"#.....#.....#", "#...........#", "#############"}, 0, 0, true},
		{"too short for the player", []string{"#...........#"}, 0, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := level.ParseRows(tile, tt.rows...)
			if err != nil {
				t.Fatal(err)
			}
			w, err := NewWorldInChunk(testTuning(), m)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("got player at (%d, %d), want an error", w.Player.Body.X, w.Player.Body.Y)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if b := w.Player.Body; b.X != tt.wantX || b.Y != tt.wantY {
				t.Fatalf("player at (%d, %d), want (%d, %d)", b.X, b.Y, tt.wantX, tt.wantY)
			}
		})
	}
}

func TestSetTuningAppliesMovement(t *testing.T) {
	w := standingWorld(t)
	tun := testTuning()
	tun.Player.RunSpeed = 90
	if err := w.SetTuning(tun); err != nil {
		t.Fatal(err)
	}
	run(w, Input{Right: true}, Hz/2)
	if w.Player.VX != 90 {
		t.Fatalf("VX = %v after the new tuning, want 90", w.Player.VX)
	}
	if w.Tuning().Player.RunSpeed != 90 {
		t.Fatal("Tuning() does not report the new tuning")
	}
}

func TestSetTuningResizesAroundFeet(t *testing.T) {
	tests := []struct {
		name         string
		w, h         int
		wantX, wantY int
	}{
		{"grow", 16, 24, 94, floorY - 24},
		{"shrink", 8, 10, 98, floorY - 10},
		// The left edge moves by -1.5 px: X 94 plus a 0.5 px remainder.
		{"odd difference", 15, 20, 94, floorY - 20},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := standingWorld(t) // 12x20 at x 96
			tun := testTuning()
			tun.Player.Width, tun.Player.Height = tt.w, tt.h
			if err := w.SetTuning(tun); err != nil {
				t.Fatal(err)
			}
			b := w.Player.Body
			if b.X != tt.wantX || b.Y != tt.wantY || b.W != tt.w || b.H != tt.h {
				t.Fatalf("body %dx%d at (%d, %d), want %dx%d at (%d, %d)",
					b.W, b.H, b.X, b.Y, tt.w, tt.h, tt.wantX, tt.wantY)
			}
			w.Step(Input{})
			if !w.Player.OnGround {
				t.Fatal("resized player is no longer standing on the floor")
			}
		})
	}
}

func TestSetTuningKeepsSizeWhenBlocked(t *testing.T) {
	tun := testTuning()
	m := testRoom(t)
	// A ceiling whose underside is two tiles (32 px) above the floor.
	for c := 1; c < 12; c++ {
		m.Set(c, 8, level.Solid)
	}
	w := NewWorld(tun, m, 96, floorY-tun.Player.Height)
	w.Step(Input{})

	tall := testTuning()
	tall.Player.Height = 40
	tall.Player.RunSpeed = 90
	if err := w.SetTuning(tall); err == nil {
		t.Fatal("growing into the ceiling did not report an error")
	}
	if b := w.Player.Body; b.H != 20 || b.Y != floorY-20 {
		t.Fatalf("body %dx%d at y %d, want the old 12x20 at y %d", b.W, b.H, b.Y, floorY-20)
	}
	if got := w.Tuning().Player; got.Height != 20 || got.RunSpeed != 90 {
		t.Fatalf("tuning height %d run %v, want the kept height 20 and the new run speed 90",
			got.Height, got.RunSpeed)
	}
}

func TestSetTuningResizeRoundTripDoesNotDrift(t *testing.T) {
	w := standingWorld(t) // 12x20 at x 96
	for _, width := range []int{13, 14, 12, 13, 12} {
		tun := testTuning()
		tun.Player.Width = width
		if err := w.SetTuning(tun); err != nil {
			t.Fatal(err)
		}
	}
	if w.Player.Body.X != 96 {
		t.Fatalf("X = %d after resizing back to 12, want 96", w.Player.Body.X)
	}
}

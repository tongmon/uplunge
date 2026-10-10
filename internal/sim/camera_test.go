package sim

import (
	"math"
	"testing"

	"github.com/tongmon/uplunge/internal/level"
)

// floorTower returns a 13x200-tile map with a solid bottom row.
func floorTower() *level.TileMap {
	m := level.NewTileMap(13, 200, tile)
	for c := 0; c < 13; c++ {
		m.Set(c, 199, level.Solid)
	}
	return m
}

// centreOnScreen returns where the player's centre is on screen, as a
// fraction of the view height from the top.
func centreOnScreen(w *World) float64 {
	b := w.Player.Body
	return (float64(b.Y) + float64(b.H)/2 - w.Camera.Y) / ViewHeight
}

func TestCameraStartsOnItsTarget(t *testing.T) {
	w := NewWorld(testTuning(), floorTower(), 96, 1600)
	if got := centreOnScreen(w); math.Abs(got-testTuning().Camera.Anchor) > 1e-9 {
		t.Fatalf("player centre at %v of the view, want %v", got, testTuning().Camera.Anchor)
	}
}

func TestCameraClosesTheGapAtRemainPerSecond(t *testing.T) {
	w := NewWorld(testTuning(), floorTower(), 96, 1600)
	// Move the player 100 px up and follow for one second of steps without
	// moving the player.
	w.Player.Body.Y -= 100
	target := w.cameraTarget()
	for i := 0; i < Hz; i++ {
		w.stepCamera()
	}
	gap := w.Camera.Y - target
	if want := 100 * testTuning().Camera.RemainPerSecond; math.Abs(gap-want) > 1e-9 {
		t.Fatalf("gap after 1 s = %v px, want %v", gap, want)
	}
}

func TestCameraOnlyMovesUp(t *testing.T) {
	w := NewWorld(testTuning(), floorTower(), 96, 1600)
	y := w.Camera.Y
	for i := 0; i < 3*Hz; i++ {
		w.Step(Input{}) // the player falls to the floor
		if w.Camera.Y > y {
			t.Fatalf("step %d: camera moved down from %v to %v", i, y, w.Camera.Y)
		}
		y = w.Camera.Y
	}
	if w.Player.Body.Y < 2400 {
		t.Fatalf("player fell only to y %d", w.Player.Body.Y)
	}
}

func TestCameraLooksAheadWhileRising(t *testing.T) {
	w := NewWorld(testTuning(), floorTower(), 96, 1600)
	still := w.cameraTarget()
	w.Player.VY = -240
	ahead := w.cameraTarget()
	if want := 240 * testTuning().Camera.Lookahead; math.Abs(still-ahead-want) > 1e-9 {
		t.Fatalf("target moved up %v px at VY -240, want %v", still-ahead, want)
	}
	w.Player.VY = 240
	if w.cameraTarget() != still {
		t.Fatal("target looks ahead while falling")
	}
}

func TestCameraStaysInsideTheMap(t *testing.T) {
	top := NewWorld(testTuning(), floorTower(), 96, 0)
	if top.Camera.Y != 0 {
		t.Fatalf("camera Y = %v with the player at the top, want 0", top.Camera.Y)
	}
	bottom, err := NewWorldInTower(testTuning(), floorTower())
	if err != nil {
		t.Fatal(err)
	}
	if want := float64(200*tile - ViewHeight); bottom.Camera.Y != want {
		t.Fatalf("camera Y = %v with the player at the bottom, want %v", bottom.Camera.Y, want)
	}
	short := NewWorld(testTuning(), testRoom(t), 96, 100)
	if short.Camera.Y != 0 {
		t.Fatalf("camera Y = %v in a map shorter than the view, want 0", short.Camera.Y)
	}
}

func TestCameraFollowsAClimb(t *testing.T) {
	// While the player rises on the whole magazine, the camera keeps the
	// player on screen. After the peak the player may fall out of view:
	// the camera never moves down.
	w := tallWorld()
	w.Step(Input{})
	for i := 0; i < 2*Hz; i++ {
		w.Step(Input{Button: true})
		if w.Player.VY >= 0 && w.Player.Fuel == 0 {
			break
		}
		if f := centreOnScreen(w); f < 0 || f > 1 {
			t.Fatalf("step %d: rising player off screen at %v of the view", i, f)
		}
	}
}

func TestNewWorldInTower(t *testing.T) {
	w, err := NewWorldInTower(testTuning(), floorTower())
	if err != nil {
		t.Fatal(err)
	}
	b := w.Player.Body
	if b.X != 98 || b.Y+b.H != 199*tile {
		t.Fatalf("player at (%d, %d), want standing at the bottom centre (98, feet %d)", b.X, b.Y, 199*tile)
	}
	open := level.NewTileMap(13, 30, tile)
	if _, err := NewWorldInTower(testTuning(), open); err == nil {
		t.Fatal("a tower without a floor under the start gave no error")
	}
}

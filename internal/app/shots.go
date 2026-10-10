package app

import (
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/tongmon/uplunge/internal/render"
)

// ParseShotTicks parses a comma-separated list of ticks such as "0,30,120"
// into a sorted list without duplicates.
func ParseShotTicks(s string) ([]uint64, error) {
	var ticks []uint64
	for _, f := range strings.Split(s, ",") {
		t, err := strconv.ParseUint(strings.TrimSpace(f), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("app: bad shot tick %q: want a non-negative integer", f)
		}
		if slices.Contains(ticks, t) {
			return nil, fmt.Errorf("app: shot tick %d listed twice", t)
		}
		ticks = append(ticks, t)
	}
	slices.Sort(ticks)
	return ticks, nil
}

// shooter saves the world at chosen ticks. The world is drawn into its own
// image inside Update, not taken from Draw, because Ebitengine may run
// several Updates between two Draws and a tick would be missed.
type shooter struct {
	dir   string
	ticks []uint64 // ascending; shot ticks are removed
	img   *ebiten.Image
}

func newShooter(dir string, ticks []uint64) (*shooter, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("app: %w", err)
	}
	return &shooter{dir: dir, ticks: ticks, img: ebiten.NewImage(ScreenWidth, ScreenHeight)}, nil
}

func (s *shooter) done() bool { return len(s.ticks) == 0 }

// maybeShoot saves the world if g is at the next chosen tick.
func (s *shooter) maybeShoot(g *game) error {
	if s.done() || g.world.Tick != s.ticks[0] {
		return nil
	}
	s.ticks = s.ticks[1:]

	s.img.Clear()
	render.World(s.img, g.world, &g.fx)
	b := s.img.Bounds()
	px := make([]byte, 4*b.Dx()*b.Dy())
	s.img.ReadPixels(px)
	rgba := &image.RGBA{Pix: px, Stride: 4 * b.Dx(), Rect: b}

	path := filepath.Join(s.dir, fmt.Sprintf("tick_%06d.png", g.world.Tick))
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("app: %w", err)
	}
	if err := png.Encode(f, rgba); err != nil {
		f.Close()
		return fmt.Errorf("app: %s: %w", path, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("app: %w", err)
	}
	log.Printf("saved %s", path)
	return nil
}

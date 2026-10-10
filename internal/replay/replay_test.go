package replay

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tongmon/uplunge/internal/level"
	"github.com/tongmon/uplunge/internal/sim"
	"github.com/tongmon/uplunge/internal/tuning"
)

var update = flag.Bool("update", false, "rewrite the .golden files from the current simulation")

var (
	none   = sim.Input{}
	right  = sim.Input{Right: true}
	leftB  = sim.Input{Left: true, Button: true}
	allIn  = sim.Input{Left: true, Right: true, Button: true}
	button = sim.Input{Button: true}
)

func TestWrite(t *testing.T) {
	r := Replay{Chunk: "Start", Inputs: []sim.Input{none, none, right, right, leftB, none}}
	var buf bytes.Buffer
	if err := Write(&buf, r); err != nil {
		t.Fatal(err)
	}
	want := "uplunge-replay 1\nchunk Start\nframes 6\n0 -\n2 R\n4 LB\n5 -\n"
	if got := buf.String(); got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
}

func TestRoundTrip(t *testing.T) {
	tests := [][]sim.Input{
		{},
		{none},
		{allIn},
		{button, button, none, right, right, leftB, allIn, none},
	}
	for i, inputs := range tests {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			var buf bytes.Buffer
			if err := Write(&buf, Replay{Chunk: "C", Inputs: inputs}); err != nil {
				t.Fatal(err)
			}
			got, err := Read(&buf)
			if err != nil {
				t.Fatal(err)
			}
			if got.Chunk != "C" || len(got.Inputs) != len(inputs) {
				t.Fatalf("got chunk %q with %d inputs, want C with %d", got.Chunk, len(got.Inputs), len(inputs))
			}
			for j := range inputs {
				if got.Inputs[j] != inputs[j] {
					t.Fatalf("input %d = %+v, want %+v", j, got.Inputs[j], inputs[j])
				}
			}
		})
	}
}

func TestReadHandWritten(t *testing.T) {
	src := "uplunge-replay 1\n\n# comment\nchunk  Shaft \nframes 5\n  2 B\n# another\n4 -\n"
	r, err := Read(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	want := []sim.Input{none, none, button, button, none}
	if r.Chunk != "Shaft" || len(r.Inputs) != len(want) {
		t.Fatalf("got %+v", r)
	}
	for i := range want {
		if r.Inputs[i] != want[i] {
			t.Fatalf("input %d = %+v, want %+v", i, r.Inputs[i], want[i])
		}
	}
}

func TestReadErrors(t *testing.T) {
	head := "uplunge-replay 1\nchunk A\nframes 10\n"
	tests := []struct {
		name    string
		src     string
		wantErr string
	}{
		{"empty", "", "header"},
		{"wrong magic", "uplunge-replay 2\n", "header"},
		{"no chunk", "uplunge-replay 1\nframes 3\n", "chunk <name>"},
		{"no frames", "uplunge-replay 1\nchunk A\n0 -\n", "frames <count>"},
		{"negative frames", "uplunge-replay 1\nchunk A\nframes -1\n", "frames <count>"},
		{"too many frames", "uplunge-replay 1\nchunk A\nframes 9223372036854775807\n", "at most"},
		{"step not increasing", head + "3 R\n3 L\n", "must increase"},
		{"step out of range", head + "10 R\n", "below 10"},
		{"bad step", head + "x R\n", "must increase"},
		{"bad input", head + "0 X\n", "bad input"},
		{"wrong order", head + "0 BL\n", "bad input"},
		{"extra field", head + "0 R B\n", "<step> <input>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Read(strings.NewReader(tt.src))
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestWriteRejectsBadChunkName(t *testing.T) {
	for _, name := range []string{"", "two words"} {
		if err := Write(&bytes.Buffer{}, Replay{Chunk: name}); err == nil {
			t.Errorf("Write with chunk %q succeeded, want an error", name)
		}
	}
}

// trace replays r against the testdata tuning and chunks, the same way the
// game sets up a run, and records the player before every step and at the end,
// so a change of even one step in timing shows up.
func trace(t *testing.T, r Replay) string {
	t.Helper()
	tun, err := tuning.Load("testdata/tuning.json")
	if err != nil {
		t.Fatal(err)
	}
	chunks, err := level.LoadLDtk("testdata/chunks.ldtk")
	if err != nil {
		t.Fatal(err)
	}
	w, _, err := Start(r, tun, chunks)
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	line := func() {
		p := w.Player
		fmt.Fprintf(&b, "tick %d x %d y %d vx %g vy %g ground %t fuel %d bullets %d tiles %d cam %.3f hp %d enemies %d over %t water %s\n",
			w.Tick, p.Body.X, p.Body.Y, p.VX, p.VY, p.OnGround, p.Fuel, len(w.Bullets), countTiles(w.Map), w.Camera.Y,
			p.HP, len(w.Enemies), w.Over, water(w))
	}
	for _, in := range r.Inputs {
		line()
		w.Step(in)
	}
	line()
	return b.String()
}

// TestGoldenReplays plays every testdata/*.rpl and compares the player's
// trajectory with the matching .golden file. Run with -update after an
// intended behavior change, and review the diff.
func TestGoldenReplays(t *testing.T) {
	paths, err := filepath.Glob("testdata/*.rpl")
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("no testdata/*.rpl files")
	}
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			r, err := Load(path)
			if err != nil {
				t.Fatal(err)
			}
			got := trace(t, r)
			if again := trace(t, r); again != got {
				t.Fatalf("two runs of the same replay differ:\n%s\nvs\n%s", got, again)
			}

			golden := strings.TrimSuffix(path, ".rpl") + ".golden"
			if *update {
				if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("%v (run go test ./internal/replay/ -update to create it)", err)
			}
			if got != string(want) {
				t.Fatalf("trajectory changed:\ngot\n%s\nwant\n%s", got, want)
			}
		})
	}
}

func TestSaveKeepsOldFileOnError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "run.rpl")
	old := Replay{Chunk: "Start", Inputs: []sim.Input{right}}
	if err := Save(path, old); err != nil {
		t.Fatal(err)
	}
	if err := Save(path, Replay{Chunk: "two words"}); err == nil {
		t.Fatal("Save with a bad chunk name succeeded")
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("old replay unreadable after failed save: %v", err)
	}
	if got.Chunk != "Start" || len(got.Inputs) != 1 || got.Inputs[0] != right {
		t.Fatalf("old replay changed after failed save: %+v", got)
	}
	if entries, _ := os.ReadDir(filepath.Dir(path)); len(entries) != 1 {
		t.Fatalf("leftover files after failed save: %v", entries)
	}
}

func TestFingerprintsRoundTrip(t *testing.T) {
	r := Replay{Chunk: "Start", Tuning: "aaaa", Map: "bbbb", Inputs: []sim.Input{right}}
	var buf bytes.Buffer
	if err := Write(&buf, r); err != nil {
		t.Fatal(err)
	}
	want := "uplunge-replay 1\nchunk Start\ntuning aaaa\nmap bbbb\nframes 1\n0 R\n"
	if buf.String() != want {
		t.Fatalf("got\n%s\nwant\n%s", buf.String(), want)
	}
	got, err := Read(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if got.Tuning != "aaaa" || got.Map != "bbbb" {
		t.Fatalf("fingerprints = %q, %q; want aaaa, bbbb", got.Tuning, got.Map)
	}
}

func TestReadFingerprintErrors(t *testing.T) {
	tests := []struct{ name, src, wantErr string }{
		{"empty tuning", "uplunge-replay 1\nchunk A\ntuning \nframes 1\n", "frames <count>"},
		{"spaced map", "uplunge-replay 1\nchunk A\nmap a b\nframes 1\n", "map <fingerprint>"},
		{"map before tuning", "uplunge-replay 1\nchunk A\nmap b\ntuning a\nframes 1\n", "frames <count>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Read(strings.NewReader(tt.src))
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestMismatches(t *testing.T) {
	tests := []struct {
		name          string
		tuning, mapFP string
		want          []string
	}{
		{"both match", "t1", "m1", nil},
		{"tuning changed", "t2", "m1", []string{"tuning differs"}},
		{"map changed", "t1", "m2", []string{`chunk "Start" differs`}},
		{"both changed", "t2", "m2", []string{"tuning differs", `chunk "Start" differs`}},
	}
	r := Replay{Chunk: "Start", Tuning: "t1", Map: "m1"}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := r.Mismatches(tt.tuning, tt.mapFP)
			if len(got) != len(tt.want) {
				t.Fatalf("got %q, want %d messages", got, len(tt.want))
			}
			for i := range got {
				if !strings.Contains(got[i], tt.want[i]) {
					t.Errorf("message %d = %q, want it to contain %q", i, got[i], tt.want[i])
				}
			}
		})
	}
	if got := (Replay{Chunk: "Start"}).Mismatches("t", "m"); got != nil {
		t.Errorf("hand-written replay without fingerprints warned: %q", got)
	}
}

// countTiles counts the non-empty cells, so broken blocks show in a trace.
func countTiles(m *level.TileMap) int {
	n := 0
	for r := 0; r < m.Rows; r++ {
		for c := 0; c < m.Cols; c++ {
			if m.At(c, r) != level.Empty {
				n++
			}
		}
	}
	return n
}

func TestTowerHeader(t *testing.T) {
	r := Replay{Tower: true, Seed: 18446744073709551615, Inputs: []sim.Input{right}}
	var buf bytes.Buffer
	if err := Write(&buf, r); err != nil {
		t.Fatal(err)
	}
	want := "uplunge-replay 1\ntower 18446744073709551615\nframes 1\n0 R\n"
	if got := buf.String(); got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
	got, err := Read(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Tower || got.Seed != r.Seed || got.Chunk != "" {
		t.Fatalf("read back %+v, want tower %d", got, r.Seed)
	}
}

func TestTowerHeaderErrors(t *testing.T) {
	for _, src := range []string{
		"uplunge-replay 1\ntower\nframes 0\n",
		"uplunge-replay 1\ntower -1\nframes 0\n",
		"uplunge-replay 1\ntower 18446744073709551616\nframes 0\n",
		"uplunge-replay 1\ntower x\nframes 0\n",
	} {
		if _, err := Read(strings.NewReader(src)); err == nil || !strings.Contains(err.Error(), "tower <seed>") {
			t.Errorf("Read(%q) error = %v, want one about the tower seed", src, err)
		}
	}
	if err := Write(&bytes.Buffer{}, Replay{Tower: true, Chunk: "Start"}); err == nil {
		t.Error("Write of a tower run with a chunk succeeded, want an error")
	}
}

func TestStart(t *testing.T) {
	tun, err := tuning.Load("testdata/tuning.json")
	if err != nil {
		t.Fatal(err)
	}
	chunks, err := level.LoadLDtk("testdata/chunks.ldtk")
	if err != nil {
		t.Fatal(err)
	}
	a, ma, err := Start(Replay{Tower: true, Seed: 5}, tun, chunks)
	if err != nil {
		t.Fatal(err)
	}
	_, mb, _ := Start(Replay{Tower: true, Seed: 5}, tun, chunks)
	_, mc, _ := Start(Replay{Tower: true, Seed: 6}, tun, chunks)
	if ma.Fingerprint() != mb.Fingerprint() {
		t.Fatal("the same seed built different towers")
	}
	if ma.Fingerprint() == mc.Fingerprint() {
		t.Fatal("seeds 5 and 6 built the same tower")
	}
	if b := a.Player.Body; b.Y+b.H != (ma.Rows-1)*ma.TileSize {
		t.Fatalf("tower run starts with feet at %d, want on the bottom floor at %d", b.Y+b.H, (ma.Rows-1)*ma.TileSize)
	}
	if _, _, err := Start(Replay{Chunk: "Nope"}, tun, chunks); err == nil {
		t.Fatal("unknown chunk gave no error")
	}
}

// water formats the water surface for a trace, or "off" without water.
func water(w *sim.World) string {
	if !w.Water.On {
		return "off"
	}
	return fmt.Sprintf("%.3f", w.Water.Y)
}

func TestLabHeader(t *testing.T) {
	r := Replay{Lab: true, Seed: 7, Inputs: []sim.Input{right}}
	var buf bytes.Buffer
	if err := Write(&buf, r); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "\nlab 7\n") {
		t.Fatalf("header %q, want a lab line", buf.String())
	}
	got, err := Read(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Lab || got.Tower || got.Seed != 7 || got.Where() != "lab 7" {
		t.Fatalf("read back %+v", got)
	}
	if err := Write(&bytes.Buffer{}, Replay{Tower: true, Lab: true}); err == nil {
		t.Fatal("a run that is both a tower and a lab was written")
	}
	if _, err := Read(strings.NewReader("uplunge-replay 1\nlab x\nframes 0\n")); err == nil {
		t.Fatal("a bad lab seed was read")
	}
}

func TestStartLab(t *testing.T) {
	tun, err := tuning.Load("testdata/tuning.json")
	if err != nil {
		t.Fatal(err)
	}
	w, m, err := Start(Replay{Lab: true, Seed: 3}, tun, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !w.Water.On || !w.Goal || len(w.Enemies) == 0 || m.Rows != tun.Lab.Rows {
		t.Fatalf("lab run: water %v goal %v enemies %d rows %d", w.Water.On, w.Goal, len(w.Enemies), m.Rows)
	}
	if gap := sim.EnemyGap(tun, m); gap < float64(tun.Lab.Spacing)-2 || gap > float64(tun.Lab.Spacing)+2 {
		t.Fatalf("lab enemy gap %v, want about the spacing %d", gap, tun.Lab.Spacing)
	}
}

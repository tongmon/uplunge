// Package replay reads and writes recorded per-step inputs. It must not depend
// on Ebitengine.
//
// The format is line-based text so replays diff well and can be written by
// hand:
//
//	uplunge-replay 1
//	chunk Start
//	frames 240
//	0 -
//	30 R
//	40 RB
//
// After the header, each line is a step index and the input from that step
// on, written only when the input changes. Inputs are any of L, R, B (left,
// right, button) in that order, or "-" for none. Steps before the first line
// have no input. Blank lines and lines starting with # are ignored.
package replay

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/tongmon/uplunge/internal/sim"
)

const magic = "uplunge-replay 1"

// Replay is a recorded run: where it started and the input for every step.
type Replay struct {
	// Chunk is the name of the chunk the run started in.
	Chunk string
	// Inputs holds one input per step.
	Inputs []sim.Input
}

// Write encodes r.
func Write(w io.Writer, r Replay) error {
	if r.Chunk == "" || strings.ContainsAny(r.Chunk, " \t\r\n") {
		return fmt.Errorf("replay: invalid chunk name %q", r.Chunk)
	}
	bw := bufio.NewWriter(w)
	fmt.Fprintf(bw, "%s\nchunk %s\nframes %d\n", magic, r.Chunk, len(r.Inputs))
	var prev sim.Input
	for i, in := range r.Inputs {
		if i == 0 || in != prev {
			fmt.Fprintf(bw, "%d %s\n", i, formatInput(in))
		}
		prev = in
	}
	return bw.Flush()
}

// Read decodes a replay written by Write or by hand.
func Read(rd io.Reader) (Replay, error) {
	sc := bufio.NewScanner(rd)
	line := 0
	next := func() (string, bool) {
		for sc.Scan() {
			line++
			if s := strings.TrimSpace(sc.Text()); s != "" && !strings.HasPrefix(s, "#") {
				return s, true
			}
		}
		return "", false
	}
	errorf := func(format string, args ...any) (Replay, error) {
		return Replay{}, fmt.Errorf("replay: line %d: %s", line, fmt.Sprintf(format, args...))
	}

	if s, ok := next(); !ok || s != magic {
		return errorf("want %q header, got %q", magic, s)
	}
	s, _ := next()
	name, ok := strings.CutPrefix(s, "chunk ")
	name = strings.TrimSpace(name)
	if !ok || name == "" {
		return errorf("want \"chunk <name>\", got %q", s)
	}
	s, _ = next()
	n, err := strconv.Atoi(strings.TrimPrefix(s, "frames "))
	if !strings.HasPrefix(s, "frames ") || err != nil || n < 0 {
		return errorf("want \"frames <count>\", got %q", s)
	}
	r := Replay{Chunk: name, Inputs: make([]sim.Input, n)}

	// cur is the input in effect from step "from" on; prev is the last step read.
	var cur sim.Input
	from, prev := 0, -1
	for {
		s, ok := next()
		if !ok {
			break
		}
		f := strings.Fields(s)
		if len(f) != 2 {
			return errorf("want \"<step> <input>\", got %q", s)
		}
		step, err := strconv.Atoi(f[0])
		if err != nil || step <= prev || step >= n {
			return errorf("step %q must increase and be below %d", f[0], n)
		}
		in, err := parseInput(f[1])
		if err != nil {
			return errorf("%v", err)
		}
		for i := from; i < step; i++ {
			r.Inputs[i] = cur
		}
		cur, from, prev = in, step, step
	}
	if err := sc.Err(); err != nil {
		return Replay{}, fmt.Errorf("replay: %w", err)
	}
	for i := from; i < n; i++ {
		r.Inputs[i] = cur
	}
	return r, nil
}

func formatInput(in sim.Input) string {
	var b strings.Builder
	if in.Left {
		b.WriteByte('L')
	}
	if in.Right {
		b.WriteByte('R')
	}
	if in.Button {
		b.WriteByte('B')
	}
	if b.Len() == 0 {
		return "-"
	}
	return b.String()
}

func parseInput(s string) (sim.Input, error) {
	if s == "-" {
		return sim.Input{}, nil
	}
	var in sim.Input
	rest := s
	for _, c := range []struct {
		prefix string
		flag   *bool
	}{{"L", &in.Left}, {"R", &in.Right}, {"B", &in.Button}} {
		if r, ok := strings.CutPrefix(rest, c.prefix); ok {
			*c.flag = true
			rest = r
		}
	}
	if rest != "" || s == "" {
		return sim.Input{}, fmt.Errorf("bad input %q: want any of L, R, B in that order, or -", s)
	}
	return in, nil
}

// Load reads a replay file.
func Load(path string) (Replay, error) {
	f, err := os.Open(path)
	if err != nil {
		return Replay{}, fmt.Errorf("replay: %w", err)
	}
	defer f.Close()
	r, err := Read(f)
	if err != nil {
		return Replay{}, fmt.Errorf("%s: %w", path, err)
	}
	return r, nil
}

// Save writes a replay file, replacing any existing one.
func Save(path string, r Replay) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("replay: %w", err)
	}
	if err := Write(f, r); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

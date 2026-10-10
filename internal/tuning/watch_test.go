package tuning

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// writeAt writes data to path and sets its modification time to at, so tests
// do not depend on the file system's timestamp resolution.
func writeAt(t *testing.T, path, data string, at time.Time) {
	t.Helper()
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, at, at); err != nil {
		t.Fatal(err)
	}
}

func TestWatcher(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tuning.json")
	t0 := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	writeAt(t, path, valid, t0)

	w, err := NewWatcher(path)
	if err != nil {
		t.Fatal(err)
	}
	poll := func(step string) (Tuning, bool, error) {
		t.Helper()
		tun, ok, err := w.Poll()
		t.Logf("%s: ok=%v err=%v", step, ok, err)
		return tun, ok, err
	}

	if _, ok, err := poll("unchanged"); ok || err != nil {
		t.Fatal("reported a change before the file changed")
	}

	faster := strings.Replace(valid, `"runSpeed": 180`, `"runSpeed": 200`, 1)
	writeAt(t, path, faster, t0.Add(time.Second))
	if tun, ok, err := poll("edited"); !ok || err != nil || tun.Player.RunSpeed != 200 {
		t.Fatalf("edit not reloaded: ok=%v err=%v run=%v", ok, err, tun.Player.RunSpeed)
	}
	if _, ok, err := poll("edited, polled again"); ok || err != nil {
		t.Fatal("reported the same change twice")
	}

	writeAt(t, path, `{"player": {`, t0.Add(2*time.Second))
	if _, ok, err := poll("half saved"); ok || err == nil {
		t.Fatal("broken file did not report an error")
	}
	if _, ok, err := poll("half saved, polled again"); ok || err != nil {
		t.Fatal("broken file was retried before it changed again")
	}

	writeAt(t, path, valid, t0.Add(3*time.Second))
	if tun, ok, err := poll("fixed"); !ok || err != nil || tun.Player.RunSpeed != 180 {
		t.Fatalf("fixed file not reloaded: ok=%v err=%v", ok, err)
	}

	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := poll("deleted"); ok || err == nil {
		t.Fatal("deleted file did not report an error")
	}
}

func TestNewWatcherMissingFile(t *testing.T) {
	if _, err := NewWatcher(filepath.Join(t.TempDir(), "nope.json")); err == nil {
		t.Fatal("watching a missing file did not fail")
	}
}

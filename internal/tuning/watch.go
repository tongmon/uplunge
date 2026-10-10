package tuning

import (
	"fmt"
	"os"
	"time"
)

// Watcher reloads a tuning file when its modification time or size changes.
// It polls; call Poll from the game loop at whatever rate suits.
type Watcher struct {
	path    string
	modTime time.Time
	size    int64
}

// NewWatcher starts watching path from its current state, so the first Poll
// reports nothing until the file changes.
func NewWatcher(path string) (*Watcher, error) {
	w := &Watcher{path: path}
	if _, err := w.changed(); err != nil {
		return nil, err
	}
	return w, nil
}

// Poll reloads the file if it changed since the last Poll. It returns the new
// tuning and true on a successful reload. A file that changed but does not
// parse returns an error and is not retried until it changes again, so a
// half-saved file is picked up once the editor finishes writing it.
func (w *Watcher) Poll() (Tuning, bool, error) {
	changed, err := w.changed()
	if err != nil || !changed {
		return Tuning{}, false, err
	}
	t, err := Load(w.path)
	if err != nil {
		return Tuning{}, false, err
	}
	return t, true, nil
}

func (w *Watcher) changed() (bool, error) {
	fi, err := os.Stat(w.path)
	if err != nil {
		return false, fmt.Errorf("tuning: %w", err)
	}
	if fi.ModTime().Equal(w.modTime) && fi.Size() == w.size {
		return false, nil
	}
	w.modTime, w.size = fi.ModTime(), fi.Size()
	return true, nil
}

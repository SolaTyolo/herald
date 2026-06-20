package routing

import (
	"context"
	"log/slog"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watch starts a background fsnotify watcher that reloads the table on write/create.
// Returns a stop function.
func Watch(ctx context.Context, t *Table) (func(), error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	dir := filepath.Dir(t.Path())
	if err := watcher.Add(dir); err != nil {
		_ = watcher.Close()
		return nil, err
	}

	var stopOnce sync.Once
	stop := func() {
		stopOnce.Do(func() { _ = watcher.Close() })
	}

	target := filepath.Clean(t.Path())
	go func() {
		var debounce *time.Timer
		var debounceMu sync.Mutex
		reload := func() {
			if err := t.Reload(); err != nil {
				slog.Error("providers routing reload rejected", "path", t.Path(), "err", err)
			}
		}
		for {
			select {
			case <-ctx.Done():
				stop()
				return
			case ev, ok := <-watcher.Events:
				if !ok {
					return
				}
				if filepath.Clean(ev.Name) != target {
					continue
				}
				if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) == 0 {
					continue
				}
				debounceMu.Lock()
				if debounce != nil {
					debounce.Stop()
				}
				debounce = time.AfterFunc(200*time.Millisecond, reload)
				debounceMu.Unlock()
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				slog.Warn("providers routing watch error", "err", err)
			}
		}
	}()
	return stop, nil
}

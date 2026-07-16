package config

import (
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	watcher  *fsnotify.Watcher
	path     string
	callback func(*Config)
	done     chan bool
	mu       sync.Mutex
	timer    *time.Timer
}

func NewWatcher(path string, callback func(*Config)) (*Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		watcher:  watcher,
		path:     path,
		callback: callback,
		done:     make(chan bool),
	}

	err = watcher.Add(path)
	if err != nil {
		watcher.Close()
		return nil, err
	}

	slog.Debug("config watcher started", "path", path)
	return w, nil
}

func (w *Watcher) Start() {
	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			if !strings.HasSuffix(event.Name, ".yaml") && !strings.HasSuffix(event.Name, ".yml") {
				continue
			}

			if event.Op&fsnotify.Chmod == fsnotify.Chmod && event.Op == fsnotify.Chmod {
				continue
			}

			slog.Debug("config file event detected",
				"file", event.Name,
				"op", event.Op.String(),
			)

			w.mu.Lock()
			if w.timer != nil {
				w.timer.Stop()
			}
			w.timer = time.AfterFunc(200*time.Millisecond, func() {
				slog.Info("reloading config", "path", w.path)
				cfg, err := LoadFile(w.path)
				if err != nil {
					slog.Error("config reload failed", "path", w.path, "err", err)
					return
				}
				w.callback(cfg)
			})
			w.mu.Unlock()

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			slog.Error("config watcher error", "err", err)
		case <-w.done:
			slog.Debug("config watcher stopped")
			return
		}
	}
}

func (w *Watcher) Close() error {
	w.done <- true
	return w.watcher.Close()
}

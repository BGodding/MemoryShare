package watcher

import (
	"context"
	"errors"
	"fmt"
	"github.com/fsnotify/fsnotify"
	log "go.uber.org/zap"
	"math"
	"sync"
	"time"
)

type Watcher struct {
	paths        []string
	eventForward chan fsnotify.Event
	watch        *fsnotify.Watcher
}

func Init(paths []string, c chan fsnotify.Event) (*Watcher, error) {
	if len(paths) < 1 {
		return nil, errors.New("must specify at least one path to watch")
	}

	// Create a new watcher.
	var w Watcher
	var err error
	w.paths = paths
	w.eventForward = c
	w.watch, err = fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("creating a new watcher: %s", err)
	}

	return &w, nil
}

func (w *Watcher) Run(ctx context.Context) error {
	defer w.watch.Close()
	for _, p := range w.paths {
		err := w.watch.Add(p)
		if err != nil {
			log.S().Fatal(err)
			return fmt.Errorf("%q: %s", p, err)
		}
	}
	return w.dedupLoop(ctx)
}

func (w *Watcher) AddPath(path string) error {
	// TODO
	return nil
}

func (w *Watcher) RemovePath(path string) error {
	// TODO
	return nil
}

func (w *Watcher) dedupLoop(ctx context.Context) error {
	var (
		// Wait 100ms for new events; each new event resets the timer.
		waitFor = 1000 * time.Millisecond

		// Keep track of the timers, as path → timer.
		mu     sync.Mutex
		timers = make(map[string]*time.Timer)

		// Callback we run.
		printEvent = func(e fsnotify.Event) {
			w.eventForward <- e
			log.S().Warnf("watcher: %+v", e)

			// Don't need to remove the timer if you don't have a lot of files.
			mu.Lock()
			delete(timers, e.Name)
			mu.Unlock()
		}
	)

	for {
		select {
		case <-ctx.Done():
			return nil
		// Read from Errors.
		case err, ok := <-w.watch.Errors:
			if !ok { // Channel was closed (i.e. Watcher.Close() was called).
				return err
			}
			log.S().Error(err.Error())
		// Read from Events.
		case e, ok := <-w.watch.Events:
			if !ok { // Channel was closed (i.e. Watcher.Close() was called).
				return nil
			}

			// Get timer.
			mu.Lock()
			t, ok := timers[e.Name]
			mu.Unlock()

			// No timer yet, so create one.
			if !ok {
				t = time.AfterFunc(math.MaxInt64, func() { printEvent(e) })
				t.Stop()

				mu.Lock()
				timers[e.Name] = t
				mu.Unlock()
			}

			// Reset the timer for this path, so it will start from 100ms again.
			t.Reset(waitFor)
		}
	}
}

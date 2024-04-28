package main

import (
	"context"
	"fmt"
	"github.com/fsnotify/fsnotify"
	. "github.com/saylorsolutions/modmake"
	"io/fs"
	"log"
	"path/filepath"
	"strings"
	"sync"
)

func runWatching(_base context.Context, task Task, flags *appFlags) (rerr error) {
	if flags.watchInterval <= 0 {
		return fmt.Errorf("watch interval '%s' is invalid", flags.watchInterval.String())
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to establish directory watcher: %w", err)
	}
	defer func() {
		_ = watcher.Close()
	}()

	var (
		wg           sync.WaitGroup
		base, cancel = context.WithCancel(_base)
	)
	defer cancel()
	wg.Add(1)
	watchDir := flags.watchDirectory().String()
	var watchDirs = []string{watchDir}
	if flags.watchSubdirs {
		err = filepath.Walk(watchDir, func(path string, info fs.FileInfo, err error) error {
			if path == watchDir {
				return nil
			}
			if err != nil {
				return err
			}
			if info.IsDir() {
				watchDirs = append(watchDirs, path)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to walk '%s': %w", watchDir, err)
		}
	}

	gate := newProcessGate(base, flags.watchInterval, task)
	defer func() {
		if err := gate.Stop(); err != nil {
			rerr = err
		}
	}()
	go func() {
		defer wg.Done()
		var (
			patterns = flags.watchPatterns()
			match    = func(path string) bool {
				return true
			}
		)
		if len(patterns) > 0 {
			match = func(name string) bool {
				for _, pattern := range patterns {
					pattern = strings.TrimSpace(pattern)
					matched, err := filepath.Match(pattern, filepath.Base(name))
					if err != nil {
						continue
					}
					if matched {
						return true
					}
				}
				return false
			}
		}
		if err := gate.Start(); err != nil {
			rerr = err
			return
		}
		for {
			select {
			case <-base.Done():
				rerr = gate.Stop()
				return
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if match(event.Name) {
					log.Printf("Received event: %s\n", event.String())
					if err := gate.Start(); err != nil {
						rerr = err
						return
					}
				}
			case err := <-watcher.Errors:
				rerr = fmt.Errorf("error watching directory '%s': %w", watchDir, err)
				return
			}
		}
	}()

	for _, watchDir := range watchDirs {
		if err := watcher.Add(watchDir); err != nil {
			return fmt.Errorf("failed to watch directory '%s': %w", watchDir, err)
		}
	}
	wg.Wait()
	if rerr != nil {
		return rerr
	}
	return nil
}

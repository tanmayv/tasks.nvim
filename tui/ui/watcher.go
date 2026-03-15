package ui

import (
	"log"

	"github.com/fsnotify/fsnotify"
)

// StartDBWatcher starts an fsnotify watcher on the database file.
// When the database file is modified (e.g., by the Neovim client running CLI commands),
// it sends a message to the Bubbletea program to trigger a UI reload.
func (m *Model) StartDBWatcher(dbPath string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("Warning: failed to start fsnotify watcher: %v", err)
		return
	}

	go func() {
		defer watcher.Close()
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				// If the file was written to, tell the UI to reload
				if event.Has(fsnotify.Write) {
					// We dispatch a custom message to Bubbletea
					m.program.Send(ReloadMsg{})
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("Watcher error: %v", err)
			}
		}
	}()

	err = watcher.Add(dbPath)
	if err != nil {
		log.Printf("Warning: failed to add db file to watcher: %v", err)
	}
}

type ReloadMsg struct{}

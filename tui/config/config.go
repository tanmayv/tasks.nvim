package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	DBPath      string              `json:"db_path"`
	InboxFile   string              `json:"inbox_file"`
	Directories []string            `json:"directories"`
	AutoTags    map[string][]string `json:"auto_tags"`
}

func LoadConfig() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(homeDir, ".config", "task-manager-tui", "config.json")

	// Set defaults
	config := &Config{
		DBPath:      filepath.Join(homeDir, ".local", "share", "nvim", "task_manager.db"),
		InboxFile:   filepath.Join(homeDir, "tasks", "inbox.md"),
		Directories: []string{filepath.Join(homeDir, "tasks")},
		AutoTags:    make(map[string][]string),
	}

	file, err := os.Open(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Config does not exist, return defaults
			return config, nil
		}
		return nil, err
	}
	defer file.Close()

	if err := json.NewDecoder(file).Decode(config); err != nil {
		return nil, err
	}

	// Default directories if empty in config
	if len(config.Directories) == 0 {
		config.Directories = []string{filepath.Join(homeDir, "tasks")}
	}

	// Expand ~ in paths
	config.DBPath = expandHome(config.DBPath, homeDir)
	config.InboxFile = expandHome(config.InboxFile, homeDir)
	for i, dir := range config.Directories {
		config.Directories[i] = expandHome(dir, homeDir)
	}

	return config, nil
}

func expandHome(path string, homeDir string) string {
	if len(path) > 0 && path[0] == '~' {
		return filepath.Join(homeDir, path[1:])
	}
	return path
}

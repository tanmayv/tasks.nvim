package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	DBPath    string `json:"db_path"`
	InboxFile string `json:"inbox_file"`
}

func LoadConfig() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(homeDir, ".config", "task-manager-tui", "config.json")

	// Set defaults
	config := &Config{
		DBPath:    filepath.Join(homeDir, ".local", "share", "nvim", "task_manager.db"),
		InboxFile: filepath.Join(homeDir, "tasks", "inbox.md"),
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

	// Expand ~ in paths
	config.DBPath = expandHome(config.DBPath, homeDir)
	config.InboxFile = expandHome(config.InboxFile, homeDir)

	return config, nil
}

func expandHome(path string, homeDir string) string {
	if len(path) > 0 && path[0] == '~' {
		return filepath.Join(homeDir, path[1:])
	}
	return path
}

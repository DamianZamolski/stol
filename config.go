package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	Games   map[string]*Game `json:"games"`
	Aliases map[string]int   `json:"aliases"`
	path    string
	dirty   bool
}

func configPath() (string, Error) {
	dir, error := os.UserConfigDir()
	if error != nil {
		return "", error
	}
	return filepath.Join(dir, "stol", "config.json"), nil
}

func loadConfig() (*Config, Error) {
	path, error := configPath()
	if error != nil {
		return nil, error
	}
	config := &Config{Games: map[string]*Game{}, Aliases: map[string]int{}, path: path}
	data, error := os.ReadFile(path)
	if errors.Is(error, os.ErrNotExist) {
		return config, nil
	}
	if error != nil {
		return nil, error
	}
	if error := json.Unmarshal(data, config); error != nil {
		return nil, fmt.Errorf("uszkodzony %s: %w", path, error)
	}
	if config.Games == nil {
		config.Games = map[string]*Game{}
	}
	if config.Aliases == nil {
		config.Aliases = map[string]int{}
	}
	return config, nil
}

func saveConfig(config *Config) Error {
	if !config.dirty {
		return nil
	}
	if error := os.MkdirAll(filepath.Dir(config.path), 0o755); error != nil {
		return error
	}
	data, error := json.MarshalIndent(config, "", "  ")
	if error != nil {
		return error
	}
	return os.WriteFile(config.path, append(data, '\n'), 0o644)
}

func (c *Config) remember(g *Game, alias string) {
	c.Games[strconv.Itoa(g.Id)] = g
	c.Aliases[normalize(alias)] = g.Id
	c.dirty = true
}

func normalize(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(s)), " ")
}

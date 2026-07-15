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

func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "stol", "config.json"), nil
}

func loadConfig() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}
	cfg := &Config{Games: map[string]*Game{}, Aliases: map[string]int{}, path: path}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("uszkodzony %s: %w", path, err)
	}
	if cfg.Games == nil {
		cfg.Games = map[string]*Game{}
	}
	if cfg.Aliases == nil {
		cfg.Aliases = map[string]int{}
	}
	return cfg, nil
}

func saveConfig(cfg *Config) error {
	if !cfg.dirty {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(cfg.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cfg.path, append(data, '\n'), 0o644)
}

func (c *Config) remember(g *Game, alias string) {
	c.Games[strconv.Itoa(g.ID)] = g
	c.Aliases[normalize(alias)] = g.ID
	c.dirty = true
}

func normalize(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(s)), " ")
}

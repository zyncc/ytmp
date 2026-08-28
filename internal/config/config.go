package config

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	General GeneralConfig `toml:"general"`
	Player  PlayerConfig  `toml:"player"`
	Theme   ThemeConfig   `toml:"theme"`
}

type GeneralConfig struct {
	ToggleFavorites bool `toml:"toggle_favorites"`
}

type PlayerConfig struct {
	Volume int `toml:"volume"`
}

type ThemeConfig struct {
	Primary   string `toml:"primary"`
	Secondary string `toml:"secondary"`
	Text      string `toml:"text"`
	Subtle    string `toml:"subtle"`
	Border    string `toml:"border"`
	Success   string `toml:"success"`
	Error     string `toml:"error"`
}

func Load() (*Config, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}

	dir := filepath.Join(configDir, "ytmp")
	path := filepath.Join(dir, "config.toml")

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			cfg := Default()

			if err := Save(cfg); err != nil {
				return nil, err
			}

			return cfg, nil
		}
		return nil, err
	}

	var cfg Config

	if _, err := toml.Decode(string(data), &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func Default() *Config {
	return &Config{
		General: GeneralConfig{
			ToggleFavorites: false,
		},
		Player: PlayerConfig{
			Volume: 100,
		},
		Theme: ThemeConfig{
			Primary:   "4",
			Secondary: "6",
			Text:      "7",
			Subtle:    "8",
			Border:    "0",
			Success:   "2",
			Error:     "1",
		},
	}
}

func Save(cfg *Config) error {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return err
	}

	dir := filepath.Join(configDir, "ytmp")

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	path := filepath.Join(dir, "config.toml")

	var buf bytes.Buffer

	if err := toml.NewEncoder(&buf).Encode(cfg); err != nil {
		return err
	}

	return os.WriteFile(path, buf.Bytes(), 0644)
}

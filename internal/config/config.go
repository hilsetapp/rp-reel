// Package config хранит настройки приложения.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config - настройки между запусками.
type Config struct {
	FFmpeg    string  `json:"ffmpeg,omitempty"`
	FFprobe   string  `json:"ffprobe,omitempty"`
	Preset    string  `json:"preset,omitempty"`
	OutDir    string  `json:"out_dir,omitempty"`
	LastLog   string  `json:"last_log,omitempty"`
	LastVideo string  `json:"last_video,omitempty"`
	VideoTS   float64 `json:"video_ts,omitempty"`
	Offset    float64 `json:"offset,omitempty"`
}

// Dir - папка настроек: sa-mod-manager-подобная, в пользовательской папке.
func Dir() string {
	base, err := os.UserConfigDir()
	if err != nil || base == "" {
		base = os.TempDir()
	}
	return filepath.Join(base, "rp-reel")
}

// Load читает настройки, при их отсутствии дает пустые.
func Load() *Config {
	c := &Config{OutDir: "reel-out"}
	data, err := os.ReadFile(filepath.Join(Dir(), "config.json"))
	if err != nil {
		return c
	}
	_ = json.Unmarshal(data, c)
	if c.OutDir == "" {
		c.OutDir = "reel-out"
	}
	return c
}

// Save записывает настройки.
func (c *Config) Save() error {
	if err := os.MkdirAll(Dir(), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(Dir(), "config.json"), data, 0o644)
}

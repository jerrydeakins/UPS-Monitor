package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	Server       string `json:"server"`
	PollInterval string `json:"poll_interval"`

	Shutdown ShutdownConfig `json:"shutdown"`
}

type ShutdownConfig struct {
	Enabled        bool   `json:"enabled"`
	OnBatteryDelay string `json:"on_battery_delay"`
	GracePeriod    string `json:"grace_period"`
}

func Default() Config {
	return Config{
		Server:       "10.1.100.12:3551",
		PollInterval: "5s",
		Shutdown: ShutdownConfig{
			Enabled:        true,
			OnBatteryDelay: "15s",
			GracePeriod:    "15s",
		},
	}
}

func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	dir = filepath.Join(dir, "UPS Monitor")

	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	return filepath.Join(dir, "config.json"), nil
}

func LoadDefault() (Config, error) {
	path, err := Path()
	if err != nil {
		return Config{}, err
	}

	cfg, err := Load(path)
	if err == nil {
		return cfg, nil
	}

	if !errors.Is(err, os.ErrNotExist) {
		return Config{}, err
	}

	// При первом запуске переносим старый config.json
	// из рабочего каталога приложения.
	legacyPath := "config.json"

	if _, err := os.Stat(legacyPath); err == nil {
		cfg, err := Load(legacyPath)
		if err != nil {
			return Config{}, err
		}

		if err := cfg.Save(path); err != nil {
			return Config{}, err
		}

		return cfg, nil
	}

	cfg = Default()

	if err := cfg.Save(path); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var cfg Config

	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) PollDuration() (time.Duration, error) {
	return time.ParseDuration(c.PollInterval)
}

func (c Config) OnBatteryDuration() (time.Duration, error) {
	return time.ParseDuration(c.Shutdown.OnBatteryDelay)
}

func (c Config) GraceDuration() (time.Duration, error) {
	return time.ParseDuration(c.Shutdown.GracePeriod)
}

func (c Config) Save(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	data = append(data, '\n')

	return os.WriteFile(path, data, 0644)
}

func (c Config) SaveDefault() error {
	path, err := Path()
	if err != nil {
		return err
	}

	return c.Save(path)
}

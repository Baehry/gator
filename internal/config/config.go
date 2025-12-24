package config

import (
	"os"
	"encoding/json"
	"path/filepath"
)

type Config struct {
	Db_url string
	Current_user_name string
}

func Read() (Config, error) {
	homePath, err := os.UserHomeDir()
	if err != nil {
		return Config{}, err
	}
	data, err := os.ReadFile(filepath.Join(homePath, ".gatorconfig.json"))
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (cfg Config) SetUser(name string) error {
	homePath, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	cfg.Current_user_name = name
	jsonData, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	os.WriteFile(filepath.Join(homePath, ".gatorconfig.json"), jsonData, 0644)
	return nil
}
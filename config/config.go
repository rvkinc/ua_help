package config

import (
	"fmt"
	"github.com/rvkinc/uasocial/internal/bot"
	"github.com/rvkinc/uasocial/internal/storage"

	"gopkg.in/yaml.v3"
)

// Config defines service configuration
type Config struct {
	BotConfig     *bot.Config     `yaml:"bot"`
	StorageConfig *storage.Config `yaml:"storage"`
}

// NewConfig reads config from file
func NewConfig(file []byte) (*Config, error) {
	var cfg = new(Config)

	err := yaml.Unmarshal(file, cfg)
	if err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	return cfg, nil
}

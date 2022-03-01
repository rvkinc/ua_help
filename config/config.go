package config

import (
	"fmt"
	"github.com/rvkinc/uasocial/internal/bot"

	"gopkg.in/yaml.v3"
)

// Config defines service configuration
type Config struct {
	BotConfig *bot.Config `yaml:"bot"`
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

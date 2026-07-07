package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Auth struct {
	Type     string `yaml:"type"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type Config struct {
	Version         string   `yaml:"version"`
	SpecPath        string   `yaml:"specification_path"`
	EmulatorURL     string   `yaml:"emulator_url"`
	Timeout         int      `yaml:"timeout"`
	Auth            Auth     `yaml:"auth"`
	ResourcesFilter []string `yaml:"resources_filter"`
	EndpointsFilter []string `yaml:"endpoints_filter"`
	OutputPath      string   `yaml:"output_path"`
	RetryCount      int      `yaml:"retry_count"`
	LogLevel        string   `yaml:"log_level"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	if cfg.Timeout == 0 {
		cfg.Timeout = 30
	}
	if cfg.RetryCount == 0 {
		cfg.RetryCount = 1
	}
	if cfg.OutputPath == "" {
		cfg.OutputPath = "./reports"
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
	if c.EmulatorURL == "" {
		return fmt.Errorf("emulator_url is required")
	}
	if c.SpecPath == "" {
		return fmt.Errorf("specification_path is required")
	}
	if c.Timeout < 0 {
		return fmt.Errorf("timeout must be >= 0")
	}
	return nil
}

package config

import (
	"time"

	"github.com/kelseyhightower/envconfig"
)

// Config contains configurable details for running the service
type Config struct {
	BindAddr        string        `envconfig:"BIND_ADDR"`
	HierarchyAPIURL string        `envconfig:"HIERARCHY_API_URL"`
	DbAddr          string        `envconfig:"HIERARCHY_DATABASE_ADDRESS"`
	ShutdownTimeout time.Duration `envconfig:"GRACEFUL_SHUTDOWN_TIMEOUT"`
	CodelistAPIURL  string        `envconfig:"CODE_LIST_URL"`
}

var configuration *Config

// Get configures the application and returns the configuration
func Get() (*Config, error) {
	if configuration == nil {
		configuration = &Config{
			BindAddr:        ":22600",
			HierarchyAPIURL: "http://localhost:22600",
			DbAddr:          "bolt://localhost:7687",
			ShutdownTimeout: 5 * time.Second,
			CodelistAPIURL:  "http://localhost:22400",
		}
		if err := envconfig.Process("", configuration); err != nil {
			return nil, err
		}
	}
	return configuration, nil
}

package config

import (
	"time"

	"github.com/kelseyhightower/envconfig"
)

// Config contains configurable details for running the service
type Config struct {
	BindAddr                   string        `envconfig:"BIND_ADDR"`
	HierarchyAPIURL            string        `envconfig:"HIERARCHY_API_URL"`
	ShutdownTimeout            time.Duration `envconfig:"GRACEFUL_SHUTDOWN_TIMEOUT"`
	HealthCheckInterval        time.Duration `envconfig:"HEALTHCHECK_INTERVAL"`
	HealthCheckCriticalTimeout time.Duration `envconfig:"HEALTHCHECK_CRITICAL_TIMEOUT"`
	CodelistAPIURL             string        `envconfig:"CODE_LIST_URL"`
	EnableURLRewriting         bool          `envconfig:"ENABLE_URL_REWRITING"`
}

var configuration *Config

// Get configures the application and returns the configuration
func Get() (*Config, error) {
	if configuration == nil {
		configuration = &Config{
			BindAddr:                   ":22600",
			HierarchyAPIURL:            "http://localhost:22600",
			ShutdownTimeout:            5 * time.Second,
			HealthCheckInterval:        30 * time.Second,
			HealthCheckCriticalTimeout: 90 * time.Second,
			CodelistAPIURL:             "http://localhost:22400",
			EnableURLRewriting:         false,
		}
		if err := envconfig.Process("", configuration); err != nil {
			return nil, err
		}
	}
	return configuration, nil
}

package config

import (
	"time"

	"github.com/kelseyhightower/envconfig"
)

// Config contains configurable details for running the service
type Config struct {
	BindAddr                    string        `envconfig:"BIND_ADDR"`
	HierarchyAPIURL             string        `envconfig:"HIERARCHY_API_URL"`
	ShutdownTimeout             time.Duration `envconfig:"GRACEFUL_SHUTDOWN_TIMEOUT"`
	HealthCheckInterval         time.Duration `envconfig:"HEALTHCHECK_INTERVAL"`
	HealthCheckRecoveryInterval time.Duration `envconfig:"HEALTHCHECK_RECOVERY_INTERVAL"`
	CodelistAPIURL              string        `envconfig:"CODE_LIST_URL"`
}

var configuration *Config

// Get configures the application and returns the configuration
func Get() (*Config, error) {
	if configuration == nil {
		configuration = &Config{
			BindAddr:                    ":22600",
			HierarchyAPIURL:             "http://localhost:22600",
			ShutdownTimeout:             5 * time.Second,
			HealthCheckInterval:         30 * time.Second,
			HealthCheckRecoveryInterval: 5 * time.Second,
			CodelistAPIURL:              "http://localhost:22400",
		}
		if err := envconfig.Process("", configuration); err != nil {
			return nil, err
		}
	}
	return configuration, nil
}

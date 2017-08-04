package config

import "github.com/ian-kent/gofigure"

//Config contains configurable details for running the service
type Config struct {
	BindAddr string `env:"BIND_ADDR" flag:"bind-addr" flagDesc:"The port to bind to"`
}

var configuration *Config

// Get configures the application and returns the configuration
func Get() (*Config, error) {
	if configuration != nil {
		return configuration, nil
	}

	configuration = &Config{
		BindAddr: ":22600",
	}

	if err := gofigure.Gofigure(configuration); err != nil {
		return configuration, err
	}

	return configuration, nil
}

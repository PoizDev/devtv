package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server     ServerConfig     `yaml:"server"`
	Database   DatabaseConfig   `yaml:"database"`
	Middleware MiddlewareConfig `yaml:"middleware"`
}

type ServerConfig struct {
	Port            string        `yaml:"port"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
	LogConfigPath   string        `yaml:"log_config_path"`
	EnvPath         string        `yaml:"env_path"`
}

type DatabaseConfig struct {
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `yaml:"conn_max_idle_time"`
}

type MiddlewareConfig struct {
	CircuitBreaker struct {
		Threshold uint64        `yaml:"threshold"`
		Timeout   time.Duration `yaml:"timeout"`
	} `yaml:"circuit_breaker"`
	RateLimit struct {
		Burst int `yaml:"burst"`
		Limit int `yaml:"limit"`
	} `yaml:"rate_limit"`
	RequestTimeout time.Duration `yaml:"request_timeout"`
	Cors           struct {      //'Buradan sonrası hayata geçmedi geçerse diye
		AllowedOrigins   string        `yaml:"allowed_origins"`
		AllowedMethods   string        `yaml:"allowed_methods"`
		ExposeHeaders    string        `yaml:"expose_headers"`
		AllowCredentials bool          `yaml:"allow_credentials"`
		MaxAge           time.Duration `yaml:"max_age"`
	}
}

func LoadConfig(filename string) (*Config, error) {
	buf, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	c := &Config{}
	err = yaml.Unmarshal(buf, c)
	if err != nil {
		return nil, err
	}

	return c, nil
}

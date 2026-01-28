package config

import (
	"errors"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config represents the root configuration structure for the application
type Config struct {
	Server  ServerConfig  `mapstructure:"server"`
	Storage StorageConfig `mapstructure:"storage"`
	GC      GCConfig      `mapstructure:"gc"`
	Log     LogConfig     `mapstructure:"log"`
}

// GCConfig defines the parameters for the background active expiration
type GCConfig struct {
	Enabled         bool
	Interval        time.Duration // how often to run the background check
	SamplesPerCheck int           // how many keys to check per loop
	MatchThreshold  float64       // 0.0-1.0. if expired/scanned > threshold, repeat immediately
}

// ServerConfig holds the network settings
type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port string `mapstructure:"port"`
}

// StorageConfig defines the internal structure of the storage engine
type StorageConfig struct {
	Shards uint `mapstructure:"shards"`
}

// LogConfig defines logging verbosity and output style
type LogConfig struct {
	Level  string `mapstructure:"level"`  // debug, info, warn, error
	Format string `mapstructure:"format"` // json, console
}

// Load reads the configuration from a file and overrides it with environment variables
func Load(path string) (*Config, error) {
	setDefaults()

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(path)
	viper.AddConfigPath(".")

	viper.SetEnvPrefix("MOONLIGHT")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return nil, err
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// setDefaults populates viper with fallback values if they are not provided via file or ENV
func setDefaults() {
	// Server
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", "6380")

	// Storage
	viper.SetDefault("storage.shards", 32)

	// GC
	viper.SetDefault("gc.enabled", true)
	viper.SetDefault("gc.interval", "100ms")
	viper.SetDefault("gc.sample_per_shard", 20)
	viper.SetDefault("gc.expand_threshold", 0.25)

	// Logger
	viper.SetDefault("log.level", "debug")
	viper.SetDefault("log.format", "json")
}

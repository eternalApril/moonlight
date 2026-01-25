package config

import "time"

type GCConfig struct {
	Enabled         bool
	Interval        time.Duration // how often to run the background check
	SamplesPerCheck int           // how many keys to check per loop
	MatchThreshold  float64       // 0.0-1.0. if expired/scanned > threshold, repeat immediately
}

func DefaultGCConfig() GCConfig {
	return GCConfig{
		Enabled:         true,
		Interval:        100 * time.Millisecond,
		SamplesPerCheck: 20,
		MatchThreshold:  0.25,
	}
}

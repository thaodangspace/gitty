package resources

import (
	"errors"
	"fmt"
	"runtime"
	"runtime/debug"

	appconfig "gitweb/server/internal/config"
)

// Config contains the runtime caps that are applied at startup.
type Config struct {
	MemoryLimitBytes int64
	GOMAXPROCS       int
}

// AppliedCaps reports the runtime caps that were applied.
type AppliedCaps struct {
	MemoryLimitBytes int64
	GOMAXPROCS       int
}

// FromAppConfig extracts runtime cap settings from the application config.
func FromAppConfig(cfg *appconfig.Config) Config {
	if cfg == nil || cfg.ResourceGovernor == nil {
		return Config{}
	}

	return Config{
		MemoryLimitBytes: cfg.ResourceGovernor.MemoryLimitBytes,
		GOMAXPROCS:       cfg.ResourceGovernor.GOMAXPROCS,
	}
}

// Validate verifies the runtime caps are usable.
func (c Config) Validate() error {
	if c.MemoryLimitBytes <= 0 {
		return errors.New("resource memoryLimitBytes must be positive")
	}
	if c.GOMAXPROCS <= 0 {
		return errors.New("resource gomaxprocs must be positive")
	}
	return nil
}

// ApplyRuntimeCaps validates and applies the configured runtime caps.
func ApplyRuntimeCaps(cfg Config) (AppliedCaps, error) {
	if err := cfg.Validate(); err != nil {
		return AppliedCaps{}, err
	}

	debug.SetMemoryLimit(cfg.MemoryLimitBytes)
	runtime.GOMAXPROCS(cfg.GOMAXPROCS)

	return AppliedCaps{
		MemoryLimitBytes: cfg.MemoryLimitBytes,
		GOMAXPROCS:       cfg.GOMAXPROCS,
	}, nil
}

func (c Config) String() string {
	return fmt.Sprintf("memory=%d gomaxprocs=%d", c.MemoryLimitBytes, c.GOMAXPROCS)
}

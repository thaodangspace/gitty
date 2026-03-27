package resources

import appconfig "gitweb/server/internal/config"

// FromAppConfig extracts runtime cap settings from the application config.
func FromAppConfig(cfg *appconfig.Config) Config {
	if cfg == nil || cfg.ResourceGovernor == nil {
		return Config{}
	}

	return Config{
		Enabled:          cfg.ResourceGovernor.Enabled,
		MemoryLimitBytes: cfg.ResourceGovernor.MemoryLimitBytes,
		GOMAXPROCS:       cfg.ResourceGovernor.GOMAXPROCS,
	}
}

package resources

import appconfig "gitweb/server/internal/config"

const (
	defaultMaxExpensiveInflight = 2
	defaultDegradeHighWatermark = 0.85
	defaultDegradeLowWatermark  = 0.70
	defaultRetryAfterSeconds    = 3
)

// FromAppConfig extracts runtime cap settings from the application config.
func FromAppConfig(cfg *appconfig.Config) Config {
	if cfg == nil || cfg.ResourceGovernor == nil {
		return Config{}
	}

	return Config{
		Enabled:              cfg.ResourceGovernor.Enabled,
		MemoryLimitBytes:     cfg.ResourceGovernor.MemoryLimitBytes,
		GOMAXPROCS:           cfg.ResourceGovernor.GOMAXPROCS,
		MaxExpensiveInflight: cfg.ResourceGovernor.MaxExpensiveInflight,
		DegradeHighWatermark: cfg.ResourceGovernor.DegradeHighWatermark,
		DegradeLowWatermark:  cfg.ResourceGovernor.DegradeLowWatermark,
		RetryAfterSeconds:    cfg.ResourceGovernor.RetryAfterSeconds,
	}
}

func withGovernorDefaults(cfg Config) Config {
	if cfg.MaxExpensiveInflight == 0 {
		cfg.MaxExpensiveInflight = defaultMaxExpensiveInflight
	}
	if cfg.DegradeHighWatermark == 0 {
		cfg.DegradeHighWatermark = defaultDegradeHighWatermark
	}
	if cfg.DegradeLowWatermark == 0 {
		cfg.DegradeLowWatermark = defaultDegradeLowWatermark
	}
	if cfg.RetryAfterSeconds == 0 {
		cfg.RetryAfterSeconds = defaultRetryAfterSeconds
	}
	return cfg
}

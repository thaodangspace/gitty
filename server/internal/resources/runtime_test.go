package resources

import (
	"testing"

	appconfig "gitweb/server/internal/config"
)

func TestApplyRuntimeCapsInvalidConfig(t *testing.T) {
	_, err := ApplyRuntimeCaps(Config{
		Enabled:          true,
		MemoryLimitBytes: 0,
		GOMAXPROCS:       2,
	})
	if err == nil {
		t.Fatal("expected error for invalid runtime caps config")
	}
}

func TestApplyRuntimeCapsDisabledConfigIsNoOp(t *testing.T) {
	got, err := ApplyRuntimeCaps(Config{
		Enabled:          false,
		MemoryLimitBytes: 0,
		GOMAXPROCS:       0,
	})
	if err != nil {
		t.Fatalf("ApplyRuntimeCaps() error = %v", err)
	}

	if got != (AppliedCaps{}) {
		t.Fatalf("ApplyRuntimeCaps() = %+v, want zero-value applied caps", got)
	}
}

func TestApplyRuntimeCapsValidConfig(t *testing.T) {
	got, err := ApplyRuntimeCaps(Config{
		Enabled:          true,
		MemoryLimitBytes: 1 << 30,
		GOMAXPROCS:       4,
	})
	if err != nil {
		t.Fatalf("ApplyRuntimeCaps() error = %v", err)
	}

	want := AppliedCaps{
		MemoryLimitBytes: 1 << 30,
		GOMAXPROCS:       4,
	}
	if got != want {
		t.Fatalf("ApplyRuntimeCaps() = %+v, want %+v", got, want)
	}
}

func TestFromAppConfig(t *testing.T) {
	got := FromAppConfig(nil)
	if got.Enabled {
		t.Fatalf("FromAppConfig(nil) enabled = true, want false")
	}

	cfg := &appconfig.Config{}
	got = FromAppConfig(cfg)
	if got.Enabled {
		t.Fatalf("FromAppConfig(empty config) enabled = true, want false")
	}

	cfg.ResourceGovernor = &appconfig.ResourceGovernorConfig{
		Enabled:          true,
		MemoryLimitBytes: 1 << 29,
		GOMAXPROCS:       3,
	}
	got = FromAppConfig(cfg)
	want := Config{
		Enabled:          true,
		MemoryLimitBytes: 1 << 29,
		GOMAXPROCS:       3,
	}
	if got != want {
		t.Fatalf("FromAppConfig() = %+v, want %+v", got, want)
	}
}

func TestRuntimeCapsFromAppConfigDefaultFallback(t *testing.T) {
	got, err := RuntimeCapsFromAppConfig(nil)
	if err != nil {
		t.Fatalf("RuntimeCapsFromAppConfig(nil) error = %v", err)
	}

	want := Config{
		Enabled:          true,
		MemoryLimitBytes: 1 << 30,
		GOMAXPROCS:       2,
	}
	if got != want {
		t.Fatalf("RuntimeCapsFromAppConfig(nil) = %+v, want %+v", got, want)
	}

	got, err = RuntimeCapsFromAppConfig(&appconfig.Config{})
	if err != nil {
		t.Fatalf("RuntimeCapsFromAppConfig(empty config) error = %v", err)
	}

	if got != want {
		t.Fatalf("RuntimeCapsFromAppConfig(empty config) = %+v, want %+v", got, want)
	}
}

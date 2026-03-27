package resources

import "testing"

func TestApplyRuntimeCapsInvalidConfig(t *testing.T) {
	_, err := ApplyRuntimeCaps(Config{
		MemoryLimitBytes: 0,
		GOMAXPROCS:       2,
	})
	if err == nil {
		t.Fatal("expected error for invalid runtime caps config")
	}
}

func TestApplyRuntimeCapsValidConfig(t *testing.T) {
	got, err := ApplyRuntimeCaps(Config{
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

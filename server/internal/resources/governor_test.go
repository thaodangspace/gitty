package resources

import "testing"

func TestGovernor_EntersDegradedAtHighWatermark(t *testing.T) {
	g := NewGovernor(Config{
		MaxExpensiveInflight: 1,
		DegradeHighWatermark: 0.85,
		DegradeLowWatermark:  0.70,
	})

	if got := g.Mode(); got != ModeNormal {
		t.Fatalf("initial mode = %q, want %q", got, ModeNormal)
	}

	g.UpdatePressure(0.84)
	if got := g.Mode(); got != ModeNormal {
		t.Fatalf("mode below high watermark = %q, want %q", got, ModeNormal)
	}

	g.UpdatePressure(0.85)
	if got := g.Mode(); got != ModeDegraded {
		t.Fatalf("mode at high watermark = %q, want %q", got, ModeDegraded)
	}
}

func TestGovernor_ExitsDegradedAtLowWatermark(t *testing.T) {
	g := NewGovernor(Config{
		MaxExpensiveInflight: 1,
		DegradeHighWatermark: 0.85,
		DegradeLowWatermark:  0.70,
	})

	g.UpdatePressure(0.90)
	if got := g.Mode(); got != ModeDegraded {
		t.Fatalf("mode after entering degraded = %q, want %q", got, ModeDegraded)
	}

	g.UpdatePressure(0.71)
	if got := g.Mode(); got != ModeDegraded {
		t.Fatalf("mode above low watermark = %q, want %q", got, ModeDegraded)
	}

	g.UpdatePressure(0.70)
	if got := g.Mode(); got != ModeNormal {
		t.Fatalf("mode at low watermark = %q, want %q", got, ModeNormal)
	}
}

func TestGovernor_RejectsWhenDegraded(t *testing.T) {
	g := NewGovernor(Config{
		MaxExpensiveInflight: 1,
		DegradeHighWatermark: 0.85,
		DegradeLowWatermark:  0.70,
	})

	g.UpdatePressure(0.90)

	admission := g.AdmitExpensive()
	if admission.Admitted {
		t.Fatal("expected expensive request to be rejected in degraded mode")
	}
	if admission.Reason != "degraded_mode" {
		t.Fatalf("rejection reason = %q, want %q", admission.Reason, "degraded_mode")
	}
	if admission.Release != nil {
		t.Fatal("rejected admission returned a release function")
	}
}

func TestGovernor_RejectsWhenExpensiveInflightSaturated(t *testing.T) {
	g := NewGovernor(Config{
		MaxExpensiveInflight: 1,
		DegradeHighWatermark: 0.85,
		DegradeLowWatermark:  0.70,
	})

	first := g.AdmitExpensive()
	if !first.Admitted {
		t.Fatalf("first expensive admission rejected: %+v", first)
	}

	second := g.AdmitExpensive()
	if second.Admitted {
		t.Fatal("expected saturated expensive request to be rejected")
	}
	if second.Reason != "expensive_limit_reached" {
		t.Fatalf("rejection reason = %q, want %q", second.Reason, "expensive_limit_reached")
	}

	first.Release()
}

func TestGovernor_AdmitsAndReleasesExpensiveToken(t *testing.T) {
	g := NewGovernor(Config{
		MaxExpensiveInflight: 1,
		DegradeHighWatermark: 0.85,
		DegradeLowWatermark:  0.70,
	})

	first := g.AdmitExpensive()
	if !first.Admitted {
		t.Fatalf("first expensive admission rejected: %+v", first)
	}
	if first.Release == nil {
		t.Fatal("expected admitted request to return a release function")
	}
	if first.Reason != "" {
		t.Fatalf("admitted request reason = %q, want empty", first.Reason)
	}

	first.Release()

	second := g.AdmitExpensive()
	if !second.Admitted {
		t.Fatalf("second expensive admission rejected after release: %+v", second)
	}
	if second.Release == nil {
		t.Fatal("expected second admitted request to return a release function")
	}
	second.Release()
}

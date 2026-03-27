package resources

import (
	"testing"
	"time"
)

func TestGovernor_EntersDegradedAtHighWatermark(t *testing.T) {
	g := NewGovernor(Config{
		Enabled:              true,
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
		Enabled:              true,
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
		Enabled:              true,
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
		Enabled:              true,
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
		Enabled:              true,
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

func TestGovernor_RejectsAdmissionBlockedBehindDegradeTransition(t *testing.T) {
	g := NewGovernor(Config{
		Enabled:              true,
		MaxExpensiveInflight: 1,
		DegradeHighWatermark: 0.85,
		DegradeLowWatermark:  0.70,
	})

	entered := make(chan struct{})
	release := make(chan struct{})
	g.testHookAfterPressureLock = func() {
		close(entered)
		<-release
	}

	updateDone := make(chan struct{})
	go func() {
		g.UpdatePressure(0.90)
		close(updateDone)
	}()

	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("pressure update did not reach synchronization point")
	}

	resultCh := make(chan Admission, 1)
	go func() {
		resultCh <- g.AdmitExpensive()
	}()

	select {
	case admission := <-resultCh:
		t.Fatalf("admission completed before degraded transition finished: %+v", admission)
	case <-time.After(50 * time.Millisecond):
	}

	close(release)

	select {
	case <-updateDone:
	case <-time.After(time.Second):
		t.Fatal("pressure update did not finish")
	}

	select {
	case admission := <-resultCh:
		if admission.Admitted {
			t.Fatal("expected admission waiting behind degraded transition to be rejected")
		}
		if admission.Reason != ReasonDegradedMode {
			t.Fatalf("rejection reason = %q, want %q", admission.Reason, ReasonDegradedMode)
		}
	case <-time.After(time.Second):
		t.Fatal("admission did not finish")
	}
}

func TestGovernor_DisabledDoesNotThrottleOrRejectExpensiveRequests(t *testing.T) {
	g := NewGovernor(Config{
		Enabled:              false,
		MaxExpensiveInflight: 1,
		DegradeHighWatermark: 0.85,
		DegradeLowWatermark:  0.70,
	})

	g.UpdatePressure(0.95)
	if got := g.Mode(); got != ModeNormal {
		t.Fatalf("disabled governor mode = %q, want %q", got, ModeNormal)
	}

	first := g.AdmitExpensive()
	second := g.AdmitExpensive()
	if !first.Admitted || !second.Admitted {
		t.Fatalf("disabled governor admissions = %+v / %+v, want both admitted", first, second)
	}
	if first.Release == nil || second.Release == nil {
		t.Fatal("disabled governor should still return release funcs for admitted requests")
	}

	first.Release()
	second.Release()
}

func TestNewGovernor_NormalizesInvalidConfig(t *testing.T) {
	g := NewGovernor(Config{
		Enabled:              true,
		MaxExpensiveInflight: -1,
		DegradeHighWatermark: 2,
		DegradeLowWatermark:  0.95,
		RetryAfterSeconds:    -3,
	})

	if got, want := g.cfg.MaxExpensiveInflight, defaultMaxExpensiveInflight; got != want {
		t.Fatalf("normalized max inflight = %d, want %d", got, want)
	}
	if got, want := g.cfg.DegradeHighWatermark, defaultDegradeHighWatermark; got != want {
		t.Fatalf("normalized high watermark = %v, want %v", got, want)
	}
	if got, want := g.cfg.DegradeLowWatermark, defaultDegradeLowWatermark; got != want {
		t.Fatalf("normalized low watermark = %v, want %v", got, want)
	}
	if got, want := g.cfg.RetryAfterSeconds, defaultRetryAfterSeconds; got != want {
		t.Fatalf("normalized retry-after = %d, want %d", got, want)
	}
}

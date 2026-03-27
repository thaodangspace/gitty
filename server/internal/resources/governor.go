package resources

import (
	"sync"
	"sync/atomic"
)

const (
	ReasonDegradedMode          = "degraded_mode"
	ReasonExpensiveLimitReached = "expensive_limit_reached"
)

type Mode string

const (
	ModeNormal   Mode = "normal"
	ModeDegraded Mode = "degraded"
)

type Admission struct {
	Admitted bool
	Reason   string
	Release  func()
}

type Governor struct {
	cfg Config

	mu                sync.Mutex
	mode              atomic.Int32
	expensiveInflight int

	testHookAfterPressureLock func()
}

func NewGovernor(cfg Config) *Governor {
	cfg = withGovernorDefaults(cfg)

	return &Governor{
		cfg: cfg,
	}
}

func (g *Governor) Mode() Mode {
	if g.mode.Load() == 1 {
		return ModeDegraded
	}
	return ModeNormal
}

func (g *Governor) UpdatePressure(ratio float64) {
	if !g.cfg.Enabled {
		return
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	if g.testHookAfterPressureLock != nil {
		g.testHookAfterPressureLock()
	}

	switch g.Mode() {
	case ModeNormal:
		if ratio >= g.cfg.DegradeHighWatermark {
			g.mode.Store(1)
		}
	case ModeDegraded:
		if ratio <= g.cfg.DegradeLowWatermark {
			g.mode.Store(0)
		}
	}
}

func (g *Governor) AdmitExpensive() Admission {
	if !g.cfg.Enabled {
		return Admission{
			Admitted: true,
			Release:  func() {},
		}
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	if g.Mode() == ModeDegraded {
		return Admission{Reason: ReasonDegradedMode}
	}

	if g.expensiveInflight >= g.cfg.MaxExpensiveInflight {
		return Admission{Reason: ReasonExpensiveLimitReached}
	}

	g.expensiveInflight++

	var once sync.Once
	return Admission{
		Admitted: true,
		Release: func() {
			once.Do(func() {
				g.mu.Lock()
				defer g.mu.Unlock()
				if g.expensiveInflight > 0 {
					g.expensiveInflight--
				}
			})
		},
	}
}

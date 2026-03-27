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
	cfg       Config
	mode      atomic.Int32
	expensive chan struct{}
}

func NewGovernor(cfg Config) *Governor {
	cfg = withGovernorDefaults(cfg)

	return &Governor{
		cfg:       cfg,
		expensive: make(chan struct{}, cfg.MaxExpensiveInflight),
	}
}

func (g *Governor) Mode() Mode {
	if g.mode.Load() == 1 {
		return ModeDegraded
	}
	return ModeNormal
}

func (g *Governor) UpdatePressure(ratio float64) {
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
	if g.Mode() == ModeDegraded {
		return Admission{Reason: ReasonDegradedMode}
	}

	select {
	case g.expensive <- struct{}{}:
		var once sync.Once
		return Admission{
			Admitted: true,
			Release: func() {
				once.Do(func() {
					<-g.expensive
				})
			},
		}
	default:
		return Admission{Reason: ReasonExpensiveLimitReached}
	}
}

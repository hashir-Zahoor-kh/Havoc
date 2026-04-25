package safety

import (
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
)

// BlackoutWindow is a named recurring time range during which no
// experiments may be scheduled. CronExpression uses standard 5-field cron
// syntax ("min hour dom month dow"); Duration is how long each firing
// remains active.
type BlackoutWindow struct {
	Name           string
	CronExpression string
	Duration       time.Duration
}

var cronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

// Active reports whether t falls inside this window's most recent firing.
// The scan looks back 31 days which covers every cron_expression Havoc
// supports (monthly is the coarsest).
func (w BlackoutWindow) Active(t time.Time) (bool, error) {
	sched, err := cronParser.Parse(w.CronExpression)
	if err != nil {
		return false, fmt.Errorf("parse cron %q: %w", w.CronExpression, err)
	}
	start := t.Add(-31 * 24 * time.Hour)
	next := sched.Next(start)
	var lastFire time.Time
	for !next.After(t) {
		lastFire = next
		next = sched.Next(next)
	}
	if lastFire.IsZero() {
		return false, nil
	}
	return !t.Before(lastFire) && t.Before(lastFire.Add(w.Duration)), nil
}

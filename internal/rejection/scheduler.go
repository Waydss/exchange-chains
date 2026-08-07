package rejection

import (
	"context"
	"fmt"
	"log"
	"time"
)

const moscowTimezone = "Europe/Moscow"

type expiredCleaner interface {
	CleanupExpired(ctx context.Context) error
}

// Scheduler runs expired rejection cleanup daily at 03:00 in Moscow.
type Scheduler struct {
	cleaner  expiredCleaner
	location *time.Location
	now      func() time.Time
}

// NewScheduler creates a daily cleanup scheduler using Europe/Moscow timezone data.
func NewScheduler(cleaner expiredCleaner) (*Scheduler, error) {
	location, err := time.LoadLocation(moscowTimezone)
	if err != nil {
		return nil, fmt.Errorf("load cleanup timezone %s: %w", moscowTimezone, err)
	}

	return &Scheduler{cleaner: cleaner, location: location, now: time.Now}, nil
}

// Start runs the scheduler until its context is cancelled and returns a completion channel.
func (s *Scheduler) Start(ctx context.Context) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			nextRun := s.nextRun(s.now())
			timer := time.NewTimer(time.Until(nextRun))
			select {
			case <-ctx.Done():
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				return
			case <-timer.C:
			}

			if err := s.cleaner.CleanupExpired(ctx); err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Printf("rejected item cleanup failed: %v", err)
			}
		}
	}()
	return done
}

func (s *Scheduler) nextRun(now time.Time) time.Time {
	moscowNow := now.In(s.location)
	next := time.Date(
		moscowNow.Year(),
		moscowNow.Month(),
		moscowNow.Day(),
		3,
		0,
		0,
		0,
		s.location,
	)
	if !next.After(moscowNow) {
		next = next.AddDate(0, 0, 1)
	}
	return next
}

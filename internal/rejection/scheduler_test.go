package rejection

import (
	"testing"
	"time"
)

func TestSchedulerNextRunUsesMoscowThreeAM(t *testing.T) {
	location, err := time.LoadLocation(moscowTimezone)
	if err != nil {
		t.Fatalf("load Moscow timezone: %v", err)
	}
	scheduler := &Scheduler{location: location}

	beforeThree := time.Date(2026, time.August, 6, 2, 30, 0, 0, location)
	if got, want := scheduler.nextRun(beforeThree), time.Date(2026, time.August, 6, 3, 0, 0, 0, location); !got.Equal(want) {
		t.Fatalf("next run before 03:00 = %s, want %s", got, want)
	}

	afterThree := time.Date(2026, time.August, 6, 3, 30, 0, 0, location)
	if got, want := scheduler.nextRun(afterThree), time.Date(2026, time.August, 7, 3, 0, 0, 0, location); !got.Equal(want) {
		t.Fatalf("next run after 03:00 = %s, want %s", got, want)
	}
}

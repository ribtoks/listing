package common

import (
	"testing"
	"time"
)

func TestConfirmed(t *testing.T) {
	s := &Subscriber{
		CreatedAt:   JsonTimeNow(),
		ConfirmedAt: JSONTime(time.Unix(1, 1)),
	}
	if s.Confirmed() {
		t.Errorf("Subscriber is confirmed with incorrect time")
	}
	s.ConfirmedAt = JSONTime(s.CreatedAt.Time().Add(1 * time.Second))
	if !s.Confirmed() {
		t.Errorf("Subscriber is not confirmed with correct time")
	}
}

func TestSubscribed(t *testing.T) {
	s := &Subscriber{
		CreatedAt:      JsonTimeNow(),
		UnsubscribedAt: JSONTime(time.Unix(1, 1)),
	}
	if s.Unsubscribed() {
		t.Errorf("Subscriber is unsubscribed with incorrect time")
	}
	s.UnsubscribedAt = JSONTime(s.CreatedAt.Time().Add(1 * time.Second))
	if !s.Unsubscribed() {
		t.Errorf("Subscriber is not unsubscribed with correct time")
	}
}

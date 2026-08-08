package domain

import (
	"testing"
	"time"
)

func TestRetentionPolicyCutoffBefore(t *testing.T) {
	now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	policy := RetentionPolicy{Days: 30}
	cutoff, err := policy.CutoffBefore(now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := now.AddDate(0, 0, -30)
	if !cutoff.Equal(want) {
		t.Fatalf("cutoff = %v, want %v", cutoff, want)
	}
}

func TestRetentionPolicyRejectsInvalidDays(t *testing.T) {
	if _, err := (RetentionPolicy{Days: 0}).CutoffBefore(time.Now()); err != ErrInvalidRetention {
		t.Fatalf("err = %v, want ErrInvalidRetention", err)
	}
}

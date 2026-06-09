package clusterhealth

import (
	"strings"
	"testing"
	"time"
)

func TestCheckReportsCleanErrorWhenNoAttemptRuns(t *testing.T) {
	started := time.Now()
	err := Check([]string{"127.0.0.1:1"}, 0)

	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "%!") {
		t.Fatalf("error contains fmt artifact: %v", err)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("zero-timeout check took too long: %v", elapsed)
	}
}

func TestCheckReturnsQuicklyWhenTCPClosed(t *testing.T) {
	started := time.Now()
	err := Check([]string{"127.0.0.1:1"}, time.Minute)

	if err == nil {
		t.Fatal("expected error")
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("closed TCP check took too long: %v", elapsed)
	}
}

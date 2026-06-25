package testutil

import (
	"testing"
	"time"
)

func TestCheckCassemKVReturnsQuicklyWhenTCPClosed(t *testing.T) {
	started := time.Now()
	err := CheckCassemKV([]string{"127.0.0.1:1"}, time.Minute)

	if err == nil {
		t.Fatal("expected error")
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("closed TCP check took too long: %v", elapsed)
	}
}

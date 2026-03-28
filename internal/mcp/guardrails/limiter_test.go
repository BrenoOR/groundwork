package guardrails_test

import (
	"testing"

	"github.com/groundwork-dev/groundwork/internal/mcp/guardrails"
)

func TestRateLimiter_allowsWithinBurst(t *testing.T) {
	lim := guardrails.NewRateLimiter(1, 5) // 1/s, burst of 5
	for i := 0; i < 5; i++ {
		if !lim.Allow() {
			t.Errorf("call %d should be allowed within burst", i+1)
		}
	}
}

func TestRateLimiter_blocksAfterBurst(t *testing.T) {
	lim := guardrails.NewRateLimiter(1, 3)
	for i := 0; i < 3; i++ {
		lim.Allow()
	}
	if lim.Allow() {
		t.Error("expected call to be blocked after burst exhausted")
	}
}

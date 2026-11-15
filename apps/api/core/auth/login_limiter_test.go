package auth

import "testing"

func TestLoginLimiter_BlocksAfterBurstOfFailures(t *testing.T) {
	l := NewDefaultLoginLimiter()
	for i := 0; i < 5; i++ {
		if l.Blocked("Ana") {
			t.Fatalf("blocked after %d failures, want 5 allowed", i)
		}
		l.Failed("ana") // same account regardless of case/spaces
	}
	if !l.Blocked(" ANA ") {
		t.Fatal("not blocked after 5 failures")
	}
	if l.Blocked("bob") {
		t.Fatal("another account was blocked")
	}
	l.Succeeded("ana")
	if l.Blocked("ana") {
		t.Fatal("still blocked after a successful login")
	}
}

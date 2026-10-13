package auth

import (
	"errors"
	"testing"
)

func TestTokensNeedAuthKey(t *testing.T) {
	t.Setenv("AUTH_EXP", "60")

	t.Setenv("AUTH_KEY", "key-a")
	token, err := GenerateToken("ana", 1)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	t.Setenv("AUTH_KEY", "")
	if _, err := GenerateToken("ana", 1); !errors.Is(err, ErrNoAuthKey) {
		t.Fatalf("GenerateToken() without AUTH_KEY error = %v, want ErrNoAuthKey", err)
	}
	if _, err := ParseToken(token); err == nil {
		t.Fatal("ParseToken() accepted a token without AUTH_KEY set")
	}

	t.Setenv("AUTH_KEY", "key-b")
	if _, err := ParseToken(token); err == nil {
		t.Fatal("ParseToken() accepted a token signed with another key")
	}
}

func TestGenerateTokenNeedsAuthExp(t *testing.T) {
	t.Setenv("AUTH_KEY", "key")
	t.Setenv("AUTH_EXP", "")
	if _, err := GenerateToken("ana", 1); !errors.Is(err, ErrNoAuthExp) {
		t.Fatalf("GenerateToken() without AUTH_EXP error = %v, want ErrNoAuthExp", err)
	}
}

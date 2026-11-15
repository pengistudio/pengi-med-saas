package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
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

// A token signed with the right key but another HMAC algorithm is rejected:
// parsers accept HS256 only.
func TestParsersRejectOtherAlgorithms(t *testing.T) {
	t.Setenv("AUTH_KEY", "key")
	claims := jwt.MapClaims{
		"type": "access_token", "username": "ana", "userId": 1,
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS512, claims).SignedString([]byte("key"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseToken(token); err == nil {
		t.Error("ParseToken() accepted an HS512 token")
	}
	if _, err := ValidateRefreshToken(token); err == nil {
		t.Error("ValidateRefreshToken() accepted an HS512 token")
	}
}

func TestGenerateTokenNeedsAuthExp(t *testing.T) {
	t.Setenv("AUTH_KEY", "key")
	t.Setenv("AUTH_EXP", "")
	if _, err := GenerateToken("ana", 1); !errors.Is(err, ErrNoAuthExp) {
		t.Fatalf("GenerateToken() without AUTH_EXP error = %v, want ErrNoAuthExp", err)
	}
}

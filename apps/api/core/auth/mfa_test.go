package auth

import (
	"testing"
	"time"
)

// RFC 6238 appendix B, SHA1 (secret "12345678901234567890"), last 6 digits.
func TestTOTPCode_RFC6238Vectors(t *testing.T) {
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	for unix, want := range map[int64]string{
		59:          "287082",
		1111111109:  "081804",
		1111111111:  "050471",
		1234567890:  "005924",
		2000000000:  "279037",
		20000000000: "353130",
	} {
		got, err := TOTPCode(secret, TOTPStep(time.Unix(unix, 0)))
		if err != nil || got != want {
			t.Errorf("T=%d: got %q (%v), want %q", unix, got, err, want)
		}
	}
}

func TestMatchTOTP_SkewWindow(t *testing.T) {
	secret, _ := NewTOTPSecret()
	now := time.Unix(1_800_000_015, 0)
	step := TOTPStep(now)
	for offset, ok := range map[int64]bool{-2: false, -1: true, 0: true, 1: true, 2: false} {
		code, _ := TOTPCode(secret, step+offset)
		matched, found := MatchTOTP(secret, code, now)
		if found != ok || (ok && matched != step+offset) {
			t.Errorf("offset %d: found=%v matched=%d", offset, found, matched)
		}
	}
	if _, found := MatchTOTP(secret, "12345", now); found {
		t.Error("short code matched")
	}
}

func setMFAEnv(t *testing.T) {
	t.Helper()
	t.Setenv("AUTH_KEY", "test-secret-key-for-jwt-signing-in-tests-only")
	t.Setenv("AUTH_EXP", "60")
}

// The mfa token opens nothing but /auth/login/2fa: every other parser
// rejects it.
func TestMFAToken_RejectedByEveryOtherParser(t *testing.T) {
	setMFAEnv(t)
	token, err := GenerateMFAToken(7, "hash", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseToken(token); err == nil {
		t.Error("ParseToken accepted an mfa token")
	}
	if _, err := ParseExchangeToken(token); err == nil {
		t.Error("ParseExchangeToken accepted an mfa token")
	}
	if _, err := ValidateRefreshToken(token); err == nil {
		t.Error("ValidateRefreshToken accepted an mfa token")
	}
	if _, err := ParseBackofficeAccessToken(token); err == nil {
		t.Error("ParseBackofficeAccessToken accepted an mfa token")
	}
	if _, _, err := ParsePasswordResetToken(token); err == nil {
		t.Error("ParsePasswordResetToken accepted an mfa token")
	}
	if _, err := ParseEmailVerificationToken(token); err == nil {
		t.Error("ParseEmailVerificationToken accepted an mfa token")
	}

	claims, err := ParseMFAToken(token, time.Now())
	if err != nil || claims.UserID != 7 || !claims.MatchesPassword("hash") || claims.MatchesPassword("other") {
		t.Fatalf("ParseMFAToken: %+v %v", claims, err)
	}
}

func TestParseMFAToken_RejectsOtherTokensAndExpiry(t *testing.T) {
	setMFAEnv(t)
	access, _ := GenerateToken("ana", 7)
	exchange, _ := GenerateExchangeToken("ana", 7)
	refresh, _, _ := GenerateRefreshToken("ana", 7, "fam")
	backoffice, _ := GenerateBackofficeToken("root", 7, 0)
	for name, token := range map[string]string{"access": access, "exchange": exchange, "refresh": refresh, "backoffice": backoffice} {
		if _, err := ParseMFAToken(token, time.Now()); err == nil {
			t.Errorf("%s token accepted as mfa token", name)
		}
	}

	issued := time.Now()
	token, _ := GenerateMFAToken(7, "hash", issued)
	if _, err := ParseMFAToken(token, issued.Add(MFATokenTTL+time.Second)); err == nil {
		t.Error("expired mfa token accepted")
	}
}

// Refresh tokens carry no type; a typed token (access, exchange...) presented
// as the refresh cookie is refused.
func TestValidateRefreshToken_RejectsTypedTokens(t *testing.T) {
	setMFAEnv(t)
	access, _ := GenerateToken("ana", 7)
	exchange, _ := GenerateExchangeToken("ana", 7)
	for name, token := range map[string]string{"access": access, "exchange": exchange} {
		if _, err := ValidateRefreshToken(token); err == nil {
			t.Errorf("%s token accepted as refresh token", name)
		}
	}
	refresh, _, _ := GenerateRefreshToken("ana", 7, "fam")
	if _, err := ValidateRefreshToken(refresh); err != nil {
		t.Errorf("refresh token refused: %v", err)
	}
}

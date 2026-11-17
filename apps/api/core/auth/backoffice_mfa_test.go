package auth

import (
	"testing"
	"time"
)

// Backoffice mfa tokens (login second step and enrollment) open nothing else:
// no backoffice or clinic parser accepts them, the clinic mfa parser
// neither (user IDs overlap between backoffice and clinic), and each purpose
// only parses as itself.
func TestBackofficeMFAToken_OnlyParsesAsItself(t *testing.T) {
	setMFAEnv(t)
	now := time.Now()
	for _, purpose := range []string{BackofficeMFAPurposeLogin, BackofficeMFAPurposeEnroll} {
		token, err := GenerateBackofficeMFAToken(purpose, 7, "hash", now)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ParseToken(token); err == nil {
			t.Errorf("%s: ParseToken accepted it", purpose)
		}
		if _, err := ParseBackofficeAccessToken(token); err == nil {
			t.Errorf("%s: ParseBackofficeAccessToken accepted it", purpose)
		}
		if _, err := ParseBackofficeRefreshToken(token); err == nil {
			t.Errorf("%s: ParseBackofficeRefreshToken accepted it", purpose)
		}
		if _, err := ParseMFAToken(token, now); err == nil {
			t.Errorf("%s: clinic ParseMFAToken accepted it", purpose)
		}
		other := BackofficeMFAPurposeLogin
		if purpose == BackofficeMFAPurposeLogin {
			other = BackofficeMFAPurposeEnroll
		}
		if _, err := ParseBackofficeMFAToken(token, other, now); err == nil {
			t.Errorf("%s token accepted as %s", purpose, other)
		}
		claims, err := ParseBackofficeMFAToken(token, purpose, now)
		if err != nil || claims.UserID != 7 || !claims.MatchesPassword("hash") || claims.MatchesPassword("other") {
			t.Fatalf("%s: ParseBackofficeMFAToken = %+v %v", purpose, claims, err)
		}
		if _, err := ParseBackofficeMFAToken(token, purpose, now.Add(BackofficeMFAEnrollTTL+time.Second)); err == nil {
			t.Errorf("%s: expired token accepted", purpose)
		}
	}

	clinic, _ := GenerateMFAToken(7, "hash", now)
	access, _ := GenerateBackofficeToken("root", 7, 0)
	for name, token := range map[string]string{"clinic mfa": clinic, "backoffice access": access} {
		for _, purpose := range []string{BackofficeMFAPurposeLogin, BackofficeMFAPurposeEnroll} {
			if _, err := ParseBackofficeMFAToken(token, purpose, now); err == nil {
				t.Errorf("%s token accepted as backoffice %s", name, purpose)
			}
		}
	}
}

// Login tokens live 5 minutes, enrollment tokens 15.
func TestBackofficeMFAToken_TTLByPurpose(t *testing.T) {
	setMFAEnv(t)
	now := time.Now()
	login, _ := GenerateBackofficeMFAToken(BackofficeMFAPurposeLogin, 7, "hash", now)
	enroll, _ := GenerateBackofficeMFAToken(BackofficeMFAPurposeEnroll, 7, "hash", now)
	later := now.Add(BackofficeMFALoginTTL + time.Second)
	if _, err := ParseBackofficeMFAToken(login, BackofficeMFAPurposeLogin, later); err == nil {
		t.Error("login token valid after its TTL")
	}
	if _, err := ParseBackofficeMFAToken(enroll, BackofficeMFAPurposeEnroll, later); err != nil {
		t.Errorf("enrollment token expired early: %v", err)
	}
}

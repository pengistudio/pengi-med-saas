package auth

import "testing"

func TestPortalGuestToken(t *testing.T) {
	t.Setenv("AUTH_KEY", "key")
	t.Setenv("AUTH_EXP", "60")
	token, err := GeneratePortalGuestToken(3, "email", "ana@example.com")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := ParsePortalGuestToken(token)
	if err != nil || claims != (PortalGuestClaims{TenantID: 3, Channel: "email", Contact: "ana@example.com"}) {
		t.Fatalf("ParsePortalGuestToken() = %+v, %v", claims, err)
	}
	raw, err := ParseToken(token)
	if err != nil || !IsPortalToken(raw) || IsBackofficeToken(raw) {
		t.Fatalf("a guest token must read as a portal token: %v", err)
	}
	if _, err := ParsePortalAccessToken(token); err == nil {
		t.Fatal("a guest token must not open a portal session")
	}
	if _, err := ParsePortalRefreshToken(token); err == nil {
		t.Fatal("a guest token must not refresh a portal session")
	}
	if _, err := ParseBackofficeAccessToken(token); err == nil {
		t.Fatal("a guest token must not open the backoffice")
	}
	access, _ := GeneratePortalAccessToken(3, 9)
	staff, _ := GenerateToken("ana", 1)
	for name, other := range map[string]string{"access": access, "staff": staff} {
		if _, err := ParsePortalGuestToken(other); err == nil {
			t.Fatalf("a %s token must not read as a guest token", name)
		}
	}
}

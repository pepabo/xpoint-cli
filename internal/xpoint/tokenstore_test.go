package xpoint

import (
	"errors"
	"testing"
	"time"

	"github.com/zalando/go-keyring"
)

func TestSaveLoadTokenRoundtrip(t *testing.T) {
	keyring.MockInit()

	in := &StoredToken{
		Subdomain:  "acme",
		DomainCode: "dom",
		ClientID:   "cid",
		Token: Token{
			AccessToken:  "AT",
			TokenType:    "bearer",
			RefreshToken: "RT",
			ExpiresIn:    3600,
			ExpiresAt:    time.Now().Add(time.Hour).UTC().Truncate(time.Second),
		},
	}
	if err := SaveToken(in); err != nil {
		t.Fatalf("SaveToken: %v", err)
	}

	out, err := LoadToken("acme")
	if err != nil {
		t.Fatalf("LoadToken: %v", err)
	}
	if out.AccessToken != in.AccessToken || out.RefreshToken != in.RefreshToken || out.Subdomain != in.Subdomain || out.ClientID != in.ClientID || out.DomainCode != in.DomainCode {
		t.Errorf("roundtrip mismatch: in=%+v out=%+v", in, out)
	}
}

func TestLoadToken_NotFound(t *testing.T) {
	keyring.MockInit()
	_, err := LoadToken("absent")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrTokenNotFound) {
		t.Errorf("err = %v, want ErrTokenNotFound", err)
	}
}

func TestSaveToken_RequiresSubdomain(t *testing.T) {
	keyring.MockInit()
	if err := SaveToken(&StoredToken{Token: Token{AccessToken: "x"}}); err == nil {
		t.Fatal("expected error when subdomain is empty")
	}
}

func TestDeleteToken(t *testing.T) {
	keyring.MockInit()
	in := &StoredToken{Subdomain: "acme", ClientID: "c", Token: Token{AccessToken: "AT"}}
	if err := SaveToken(in); err != nil {
		t.Fatalf("SaveToken: %v", err)
	}
	if err := DeleteToken("acme"); err != nil {
		t.Fatalf("DeleteToken: %v", err)
	}
	if _, err := LoadToken("acme"); !errors.Is(err, ErrTokenNotFound) {
		t.Errorf("expected ErrTokenNotFound after delete, got %v", err)
	}
	// Deleting a missing entry should be a no-op.
	if err := DeleteToken("acme"); err != nil {
		t.Errorf("DeleteToken on missing entry should not error, got %v", err)
	}
}

func TestLoadToken_RejectsEmptyAccessToken(t *testing.T) {
	keyring.MockInit()
	in := &StoredToken{Subdomain: "acme", ClientID: "c"}
	if err := SaveToken(in); err != nil {
		t.Fatalf("SaveToken: %v", err)
	}
	if _, err := LoadToken("acme"); err == nil {
		t.Error("expected error when stored token has no access token")
	}
}

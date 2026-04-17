package xpoint

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/zalando/go-keyring"
)

// keyringService is the service name used for all xpoint-cli entries in the
// system keyring (Secret Service on Linux, Keychain on macOS, Credential
// Manager on Windows).
const keyringService = "xpoint-cli"

// ErrTokenNotFound is returned when no token is stored for a subdomain.
var ErrTokenNotFound = errors.New("no stored token found")

// StoredToken is the persisted representation of an OAuth token together with
// the subdomain it was issued for. The client_id is captured so a subsequent
// refresh can be performed without the user re-supplying it.
type StoredToken struct {
	Subdomain  string `json:"subdomain"`
	DomainCode string `json:"domain_code,omitempty"`
	ClientID   string `json:"client_id"`
	Token
}

// SaveToken stores the token in the system keyring keyed by t.Subdomain.
func SaveToken(t *StoredToken) error {
	if t.Subdomain == "" {
		return errors.New("StoredToken.Subdomain is required")
	}
	b, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("encode token: %w", err)
	}
	if err := keyring.Set(keyringService, t.Subdomain, string(b)); err != nil {
		return fmt.Errorf("store token in keyring: %w", err)
	}
	return nil
}

// LoadToken retrieves the token stored for subdomain. Returns ErrTokenNotFound
// (matched via errors.Is) when no entry exists.
func LoadToken(subdomain string) (*StoredToken, error) {
	if subdomain == "" {
		return nil, errors.New("subdomain is required to load a token")
	}
	raw, err := keyring.Get(keyringService, subdomain)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return nil, ErrTokenNotFound
		}
		return nil, fmt.Errorf("read token from keyring: %w", err)
	}
	var t StoredToken
	if err := json.Unmarshal([]byte(raw), &t); err != nil {
		return nil, fmt.Errorf("decode token from keyring: %w", err)
	}
	if t.AccessToken == "" {
		return nil, errors.New("stored token does not contain an access token")
	}
	return &t, nil
}

// DeleteToken removes the token entry for subdomain. Missing entries are not
// treated as an error.
func DeleteToken(subdomain string) error {
	if subdomain == "" {
		return errors.New("subdomain is required to delete a token")
	}
	if err := keyring.Delete(keyringService, subdomain); err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return nil
		}
		return fmt.Errorf("delete token from keyring: %w", err)
	}
	return nil
}

package xpoint

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/zalando/go-keyring"
)

// keyringService is the service name used for the xpoint-cli entry in the
// system keyring (Secret Service on Linux, Keychain on macOS, Credential
// Manager on Windows).
const keyringService = "xpoint-cli"

// keyringAccount is the fixed account/key under which the single login
// record is stored. xpoint-cli does not support multiple concurrent logins,
// so every `xp auth login` overwrites this one entry.
const keyringAccount = "default"

// ErrTokenNotFound is returned when no login has been recorded.
var ErrTokenNotFound = errors.New("no stored token found")

// StoredToken is the persisted representation of a login: the OAuth token
// plus the subdomain, domain code, and client ID it was issued for. The
// latter three allow a later command to refresh the token and issue
// requests without the user re-supplying any flags or env vars.
type StoredToken struct {
	Subdomain  string `json:"subdomain"`
	DomainCode string `json:"domain_code,omitempty"`
	ClientID   string `json:"client_id"`
	Token
}

// SaveToken overwrites the single stored login. t.Subdomain is required
// because commands use it to build request URLs; the other fields may be
// empty if the caller has nothing to record for them.
func SaveToken(t *StoredToken) error {
	if t.Subdomain == "" {
		return errors.New("StoredToken.Subdomain is required")
	}
	b, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("encode token: %w", err)
	}
	if err := keyring.Set(keyringService, keyringAccount, string(b)); err != nil {
		return fmt.Errorf("store token in keyring: %w", err)
	}
	return nil
}

// LoadToken returns the stored login, or ErrTokenNotFound (matched via
// errors.Is) when no entry exists.
func LoadToken() (*StoredToken, error) {
	raw, err := keyring.Get(keyringService, keyringAccount)
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
	return &t, nil
}

// DeleteToken removes the stored login. Missing entries are not treated as
// an error.
func DeleteToken() error {
	if err := keyring.Delete(keyringService, keyringAccount); err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return nil
		}
		return fmt.Errorf("delete token from keyring: %w", err)
	}
	return nil
}

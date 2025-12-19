// Package config provides keychain integration for secure credential storage.
package config

import (
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/zalando/go-keyring"
)

const (
	// ServiceName is the keychain service identifier
	ServiceName = "fluxbase-cli"
)

// KeychainStore stores credentials in the system keychain
type KeychainStore struct {
	serviceName string
}

// NewKeychainStore creates a new keychain store
func NewKeychainStore() *KeychainStore {
	return &KeychainStore{
		serviceName: ServiceName,
	}
}

// IsAvailable checks if keychain is available on this system
func (k *KeychainStore) IsAvailable() bool {
	// Try a simple operation to see if keychain works
	// On systems without a keychain, this will fail
	switch runtime.GOOS {
	case "darwin", "windows":
		return true
	case "linux":
		// Linux requires a secret service (like gnome-keyring)
		// Try to detect if it's available
		err := keyring.Set(ServiceName, "__test__", "test")
		if err != nil {
			return false
		}
		_ = keyring.Delete(ServiceName, "__test__")
		return true
	default:
		return false
	}
}

// Save stores credentials in keychain
func (k *KeychainStore) Save(profileName string, creds *Credentials) error {
	data, err := json.Marshal(creds)
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	if err := keyring.Set(k.serviceName, profileName, string(data)); err != nil {
		return fmt.Errorf("failed to save to keychain: %w", err)
	}

	return nil
}

// Load retrieves credentials from keychain
func (k *KeychainStore) Load(profileName string) (*Credentials, error) {
	data, err := keyring.Get(k.serviceName, profileName)
	if err != nil {
		if err == keyring.ErrNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to load from keychain: %w", err)
	}

	var creds Credentials
	if err := json.Unmarshal([]byte(data), &creds); err != nil {
		return nil, fmt.Errorf("failed to unmarshal credentials: %w", err)
	}

	return &creds, nil
}

// Delete removes credentials from keychain
func (k *KeychainStore) Delete(profileName string) error {
	err := keyring.Delete(k.serviceName, profileName)
	if err != nil && err != keyring.ErrNotFound {
		return fmt.Errorf("failed to delete from keychain: %w", err)
	}
	return nil
}

// CredentialManager manages credentials across file and keychain storage
type CredentialManager struct {
	config   *Config
	keychain *KeychainStore
}

// NewCredentialManager creates a new credential manager
func NewCredentialManager(cfg *Config) *CredentialManager {
	return &CredentialManager{
		config:   cfg,
		keychain: NewKeychainStore(),
	}
}

// GetCredentials retrieves credentials for a profile, checking keychain if configured
func (m *CredentialManager) GetCredentials(profileName string) (*Credentials, error) {
	profile, err := m.config.GetProfile(profileName)
	if err != nil {
		return nil, err
	}

	// If using keychain storage, load from there
	if profile.CredentialStore == "keychain" {
		creds, err := m.keychain.Load(profileName)
		if err != nil {
			return nil, err
		}
		if creds != nil {
			return creds, nil
		}
		// Fall back to file if keychain is empty
	}

	// Return file-based credentials
	return profile.Credentials, nil
}

// SaveCredentials saves credentials for a profile
func (m *CredentialManager) SaveCredentials(profileName string, creds *Credentials, useKeychain bool) error {
	profile, err := m.config.GetProfile(profileName)
	if err != nil {
		return err
	}

	if useKeychain {
		if !m.keychain.IsAvailable() {
			return fmt.Errorf("keychain is not available on this system")
		}

		if err := m.keychain.Save(profileName, creds); err != nil {
			return err
		}

		profile.CredentialStore = "keychain"
		// Don't store credentials in file when using keychain
		profile.Credentials = nil
	} else {
		profile.CredentialStore = "file"
		profile.Credentials = creds
	}

	return nil
}

// DeleteCredentials removes credentials for a profile
func (m *CredentialManager) DeleteCredentials(profileName string) error {
	profile, err := m.config.GetProfile(profileName)
	if err != nil {
		return err
	}

	// Delete from keychain if it was used
	if profile.CredentialStore == "keychain" {
		if err := m.keychain.Delete(profileName); err != nil {
			return err
		}
	}

	// Clear file-based credentials
	profile.Credentials = nil

	return nil
}

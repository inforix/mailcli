package config

import (
	"fmt"
	"os"
	"path/filepath"
)

const AppName = "mailcli"

func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home dir: %w", err)
	}

	return filepath.Join(home, ".config", AppName), nil
}

func EnsureDir() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("ensure config dir: %w", err)
	}

	return dir, nil
}

// KeyringDir is where the keyring "file" backend stores encrypted entries.
func KeyringDir() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, "keyring"), nil
}

func EnsureKeyringDir() (string, error) {
	dir, err := KeyringDir()
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("ensure keyring dir: %w", err)
	}

	return dir, nil
}

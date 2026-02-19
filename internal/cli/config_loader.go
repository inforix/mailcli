package cli

import (
	"errors"
	"os"

	"mailcli/internal/config"
	"mailcli/internal/secrets"
)

func loadConfig() (config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return cfg, err
	}

	if _, ok := os.LookupEnv("MAILCLI_AUTH_PASSWORD"); ok {
		cfg.Auth.PasswordSource = "env"
		return cfg, nil
	}

	if cfg.Auth.Password != "" {
		cfg.Auth.PasswordSource = "config"
		return cfg, nil
	}

	if cfg.Auth.Username == "" {
		return cfg, nil
	}

	password, err := secrets.GetPassword(cfg.Auth.Username)
	if err != nil {
		if errors.Is(err, secrets.ErrSecretNotFound) {
			return cfg, nil
		}
		return cfg, err
	}

	cfg.Auth.Password = password
	cfg.Auth.PasswordSource = "keyring"
	return cfg, nil
}

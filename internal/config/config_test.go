package config

import (
	"testing"
)

func TestLoadConfigWithEnvOverride(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	cfg := DefaultConfig()
	cfg.IMAP.Host = "imap.example.com"
	cfg.SMTP.Host = "smtp.example.com"
	cfg.Auth.Username = "user@example.com"
	cfg.Auth.Password = "secret"

	if _, err := Save(cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	t.Setenv("MAILCLI_IMAP_HOST", "env.imap.local")

	loaded, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if loaded.IMAP.Host != "env.imap.local" {
		t.Fatalf("expected env override, got %q", loaded.IMAP.Host)
	}
	if loaded.SMTP.Host != "smtp.example.com" {
		t.Fatalf("expected smtp host from file, got %q", loaded.SMTP.Host)
	}
}

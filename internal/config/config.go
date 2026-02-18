package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

type Config struct {
	IMAP     IMAPConfig     `mapstructure:"imap" yaml:"imap"`
	SMTP     SMTPConfig     `mapstructure:"smtp" yaml:"smtp"`
	Auth     AuthConfig     `mapstructure:"auth" yaml:"auth"`
	Defaults DefaultsConfig `mapstructure:"defaults" yaml:"defaults"`
}

type IMAPConfig struct {
	Host               string `mapstructure:"host" yaml:"host"`
	Port               int    `mapstructure:"port" yaml:"port"`
	TLS                bool   `mapstructure:"tls" yaml:"tls"`
	StartTLS           bool   `mapstructure:"starttls" yaml:"starttls"`
	InsecureSkipVerify bool   `mapstructure:"insecure_skip_verify" yaml:"insecure_skip_verify"`
}

type SMTPConfig struct {
	Host               string `mapstructure:"host" yaml:"host"`
	Port               int    `mapstructure:"port" yaml:"port"`
	TLS                bool   `mapstructure:"tls" yaml:"tls"`
	StartTLS           bool   `mapstructure:"starttls" yaml:"starttls"`
	InsecureSkipVerify bool   `mapstructure:"insecure_skip_verify" yaml:"insecure_skip_verify"`
}

type AuthConfig struct {
	Username string `mapstructure:"username" yaml:"username"`
	Password string `mapstructure:"password" yaml:"password"`
}

type DefaultsConfig struct {
	DraftsMailbox string `mapstructure:"drafts_mailbox" yaml:"drafts_mailbox"`
}

func DefaultConfig() Config {
	return Config{
		IMAP: IMAPConfig{
			Port:     993,
			TLS:      true,
			StartTLS: false,
		},
		SMTP: SMTPConfig{
			Port:     587,
			TLS:      false,
			StartTLS: true,
		},
		Defaults: DefaultsConfig{
			DraftsMailbox: "Drafts",
		},
	}
}

func ConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "mailcli", "config.yaml"), nil
}

func Load() (Config, error) {
	cfg := DefaultConfig()

	path, err := ConfigPath()
	if err != nil {
		return cfg, err
	}

	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	v.SetEnvPrefix("MAILCLI")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	setDefaults(v, cfg)

	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return cfg, err
		}
	}

	if err := v.Unmarshal(&cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}

func Save(cfg Config) (string, error) {
	path, err := ConfigPath()
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", err
	}

	return path, nil
}

func Redact(cfg Config) Config {
	masked := cfg
	if masked.Auth.Password != "" {
		masked.Auth.Password = "****"
	}
	return masked
}

func setDefaults(v *viper.Viper, cfg Config) {
	v.SetDefault("imap.port", cfg.IMAP.Port)
	v.SetDefault("imap.tls", cfg.IMAP.TLS)
	v.SetDefault("imap.starttls", cfg.IMAP.StartTLS)
	v.SetDefault("imap.insecure_skip_verify", cfg.IMAP.InsecureSkipVerify)

	v.SetDefault("smtp.port", cfg.SMTP.Port)
	v.SetDefault("smtp.tls", cfg.SMTP.TLS)
	v.SetDefault("smtp.starttls", cfg.SMTP.StartTLS)
	v.SetDefault("smtp.insecure_skip_verify", cfg.SMTP.InsecureSkipVerify)

	v.SetDefault("defaults.drafts_mailbox", cfg.Defaults.DraftsMailbox)
}

func Validate(cfg Config) error {
	if err := ValidateIMAP(cfg); err != nil {
		return err
	}
	if err := ValidateSMTP(cfg); err != nil {
		return err
	}
	return nil
}

func ValidateIMAP(cfg Config) error {
	if cfg.IMAP.Host == "" {
		return fmt.Errorf("imap.host is required")
	}
	if cfg.Auth.Username == "" {
		return fmt.Errorf("auth.username is required")
	}
	if cfg.Auth.Password == "" {
		return fmt.Errorf("auth.password is required")
	}
	return nil
}

func ValidateSMTP(cfg Config) error {
	if cfg.SMTP.Host == "" {
		return fmt.Errorf("smtp.host is required")
	}
	if cfg.Auth.Username == "" {
		return fmt.Errorf("auth.username is required")
	}
	if cfg.Auth.Password == "" {
		return fmt.Errorf("auth.password is required")
	}
	return nil
}

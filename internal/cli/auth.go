package cli

import (
	"errors"
	"fmt"
	"strings"

	"mailcli/internal/config"
	"mailcli/internal/secrets"

	"github.com/spf13/cobra"
)

func newAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authentication and config setup",
	}
	cmd.AddCommand(newAuthLoginCmd())
	cmd.AddCommand(newAuthKeyringCmd())
	return cmd
}

func newAuthLoginCmd() *cobra.Command {
	var (
		imapHost     string
		imapPort     int
		imapTLS      bool
		imapStartTLS bool
		imapInsecure bool

		smtpHost     string
		smtpPort     int
		smtpTLS      bool
		smtpStartTLS bool
		smtpInsecure bool

		username      string
		password      string
		draftsMailbox string
	)

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Store IMAP/SMTP credentials and configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			passwordChanged := cmd.Flags().Changed("password")
			usernameChanged := cmd.Flags().Changed("username")

			if cmd.Flags().Changed("imap-host") {
				cfg.IMAP.Host = imapHost
			}
			if cmd.Flags().Changed("imap-port") {
				cfg.IMAP.Port = imapPort
			}
			if cmd.Flags().Changed("imap-tls") {
				cfg.IMAP.TLS = imapTLS
			}
			if cmd.Flags().Changed("imap-starttls") {
				cfg.IMAP.StartTLS = imapStartTLS
			}
			if cmd.Flags().Changed("imap-insecure") {
				cfg.IMAP.InsecureSkipVerify = imapInsecure
			}

			if cmd.Flags().Changed("smtp-host") {
				cfg.SMTP.Host = smtpHost
			}
			if cmd.Flags().Changed("smtp-port") {
				cfg.SMTP.Port = smtpPort
			}
			if cmd.Flags().Changed("smtp-tls") {
				cfg.SMTP.TLS = smtpTLS
			}
			if cmd.Flags().Changed("smtp-starttls") {
				cfg.SMTP.StartTLS = smtpStartTLS
			}
			if cmd.Flags().Changed("smtp-insecure") {
				cfg.SMTP.InsecureSkipVerify = smtpInsecure
			}

			if cmd.Flags().Changed("username") {
				cfg.Auth.Username = username
			}
			if cmd.Flags().Changed("drafts-mailbox") {
				cfg.Defaults.DraftsMailbox = draftsMailbox
			}

			if usernameChanged && !passwordChanged && (cfg.Auth.PasswordSource == "" || cfg.Auth.PasswordSource == "keyring") {
				cfg.Auth.Password = ""
				cfg.Auth.PasswordSource = ""
				if cfg.Auth.Username != "" {
					loaded, err := secrets.GetPassword(cfg.Auth.Username)
					if err != nil && !errors.Is(err, secrets.ErrSecretNotFound) {
						return err
					}
					if err == nil {
						cfg.Auth.Password = loaded
						cfg.Auth.PasswordSource = "keyring"
					}
				}
			}

			if passwordChanged {
				if password == "" {
					return fmt.Errorf("password is required")
				}
				cfg.Auth.Password = password
				cfg.Auth.PasswordSource = "flags"
			}

			if err := config.Validate(cfg); err != nil {
				return err
			}

			if passwordChanged {
				if err := secrets.SetPassword(cfg.Auth.Username, password); err != nil {
					return err
				}
			}

			if passwordChanged || cfg.Auth.PasswordSource == "keyring" || cfg.Auth.PasswordSource == "env" {
				cfg.Auth.Password = ""
			}

			path, err := config.Save(cfg)
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Config saved to %s\n", path)
			if passwordChanged {
				fmt.Fprintln(cmd.OutOrStdout(), "Password stored in keyring.")
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&imapHost, "imap-host", "", "IMAP host")
	cmd.Flags().IntVar(&imapPort, "imap-port", 0, "IMAP port")
	cmd.Flags().BoolVar(&imapTLS, "imap-tls", false, "Use IMAP TLS")
	cmd.Flags().BoolVar(&imapStartTLS, "imap-starttls", false, "Use IMAP STARTTLS")
	cmd.Flags().BoolVar(&imapInsecure, "imap-insecure", false, "Skip IMAP TLS verification")

	cmd.Flags().StringVar(&smtpHost, "smtp-host", "", "SMTP host")
	cmd.Flags().IntVar(&smtpPort, "smtp-port", 0, "SMTP port")
	cmd.Flags().BoolVar(&smtpTLS, "smtp-tls", false, "Use SMTP TLS")
	cmd.Flags().BoolVar(&smtpStartTLS, "smtp-starttls", false, "Use SMTP STARTTLS")
	cmd.Flags().BoolVar(&smtpInsecure, "smtp-insecure", false, "Skip SMTP TLS verification")

	cmd.Flags().StringVar(&username, "username", "", "Username")
	cmd.Flags().StringVar(&password, "password", "", "Password or app password")
	cmd.Flags().StringVar(&draftsMailbox, "drafts-mailbox", "", "Drafts mailbox name")

	return cmd
}

func newAuthKeyringCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "keyring [backend]",
		Short: "Show or set keyring backend",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				info, err := secrets.ResolveKeyringBackendInfo()
				if err != nil {
					return err
				}

				cfg, err := config.LoadFile()
				if err != nil {
					return err
				}

				configValue := cfg.KeyringBackend
				if strings.TrimSpace(configValue) == "" {
					configValue = "(unset)"
				}

				fmt.Fprintf(cmd.OutOrStdout(), "effective_backend: %s\nsource: %s\nconfig_backend: %s\n", info.Value, info.Source, configValue)
				if info.Source == "env" {
					fmt.Fprintln(cmd.OutOrStdout(), "note: MAILCLI_KEYRING_BACKEND overrides config")
				}
				return nil
			}

			backend := strings.ToLower(strings.TrimSpace(args[0]))
			switch backend {
			case "auto", "keychain", "file":
			default:
				return fmt.Errorf("invalid backend %q (expected auto, keychain, or file)", backend)
			}

			cfg, err := config.LoadFile()
			if err != nil {
				return err
			}

			cfg.KeyringBackend = backend

			path, err := config.Save(cfg)
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Keyring backend set to %s in %s\n", backend, path)
			return nil
		},
	}

	return cmd
}

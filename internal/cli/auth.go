package cli

import (
	"fmt"

	"mailcli/internal/config"

	"github.com/spf13/cobra"
)

func newAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authentication and config setup",
	}
	cmd.AddCommand(newAuthLoginCmd())
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
			cfg, err := config.Load()
			if err != nil {
				return err
			}

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
			if cmd.Flags().Changed("password") {
				cfg.Auth.Password = password
			}
			if cmd.Flags().Changed("drafts-mailbox") {
				cfg.Defaults.DraftsMailbox = draftsMailbox
			}

			if err := config.Validate(cfg); err != nil {
				return err
			}

			path, err := config.Save(cfg)
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Config saved to %s\n", path)
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

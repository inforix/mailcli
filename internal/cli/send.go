package cli

import (
	"fmt"

	"mailcli/internal/config"
	"mailcli/internal/email"
	"mailcli/internal/smtp"

	"github.com/spf13/cobra"
)

func newSendCmd() *cobra.Command {
	var to string
	var cc string
	var bcc string
	var subject string
	var body string
	var bodyFile string
	var attachments []string

	cmd := &cobra.Command{
		Use:   "send",
		Short: "Send an email",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if err := config.ValidateSMTP(cfg); err != nil {
				return err
			}

			content, err := loadBody(body, bodyFile)
			if err != nil {
				return err
			}

			toList := splitList(to)
			ccList := splitList(cc)
			bccList := splitList(bcc)
			recipients := append(append([]string{}, toList...), ccList...)
			recipients = append(recipients, bccList...)
			if len(recipients) == 0 {
				return fmt.Errorf("at least one recipient is required")
			}

			msg, err := email.BuildMessage(email.ComposeInput{
				From:        cfg.Auth.Username,
				To:          toList,
				Cc:          ccList,
				Bcc:         bccList,
				Subject:     subject,
				Body:        content,
				Attachments: attachments,
			})
			if err != nil {
				return err
			}

			if err := smtp.Send(cfg, cfg.Auth.Username, recipients, msg); err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Sent.")
			return nil
		},
	}

	cmd.Flags().StringVar(&to, "to", "", "Comma-separated recipients")
	cmd.Flags().StringVar(&cc, "cc", "", "Comma-separated CC recipients")
	cmd.Flags().StringVar(&bcc, "bcc", "", "Comma-separated BCC recipients")
	cmd.Flags().StringVar(&subject, "subject", "", "Message subject")
	cmd.Flags().StringVar(&body, "body", "", "Message body")
	cmd.Flags().StringVar(&bodyFile, "body-file", "", "Path to file containing message body")
	cmd.Flags().StringSliceVar(&attachments, "attachment", nil, "Attachment file paths (repeatable)")

	return cmd
}

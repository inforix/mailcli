package cli

import (
	"fmt"
	"strconv"

	"mailcli/internal/config"
	"mailcli/internal/email"
	"mailcli/internal/imap"
	"mailcli/internal/smtp"

	"github.com/spf13/cobra"
)

func newDraftCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "draft",
		Short: "Draft operations",
	}
	cmd.AddCommand(newDraftSaveCmd())
	cmd.AddCommand(newDraftListCmd())
	cmd.AddCommand(newDraftSendCmd())
	return cmd
}

func newDraftSaveCmd() *cobra.Command {
	var to string
	var cc string
	var bcc string
	var subject string
	var body string
	var bodyFile string
	var attachments []string

	cmd := &cobra.Command{
		Use:   "save",
		Short: "Save a draft to the Drafts mailbox",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if err := config.ValidateIMAP(cfg); err != nil {
				return err
			}

			content, err := loadBody(body, bodyFile)
			if err != nil {
				return err
			}

			msg, err := email.BuildMessage(email.ComposeInput{
				From:           cfg.Auth.Username,
				To:             splitList(to),
				Cc:             splitList(cc),
				Bcc:            splitList(bcc),
				Subject:        subject,
				Body:           content,
				Attachments:    attachments,
				StoreBccHeader: len(splitList(bcc)) > 0,
			})
			if err != nil {
				return err
			}

			service := imap.NewService()
			drafts := cfg.Defaults.DraftsMailbox
			if drafts == "" {
				drafts = "Drafts"
			}
			if err := service.SaveDraft(cfg, drafts, msg); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Draft saved to %s.\n", drafts)
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

func newDraftListCmd() *cobra.Command {
	var page int
	var pageSize int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List drafts",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if err := config.ValidateIMAP(cfg); err != nil {
				return err
			}

			service := imap.NewService()
			drafts := cfg.Defaults.DraftsMailbox
			if drafts == "" {
				drafts = "Drafts"
			}

			messages, total, err := service.ListMessages(cfg, drafts, page, pageSize)
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Drafts: %s (total %d)\n", drafts, total)
			printMessages(cmd.OutOrStdout(), messages)
			return nil
		},
	}

	cmd.Flags().IntVar(&page, "page", 1, "Page number (1-based, newest first)")
	cmd.Flags().IntVar(&pageSize, "page-size", 20, "Messages per page")

	return cmd
}

func newDraftSendCmd() *cobra.Command {
	var keep bool

	cmd := &cobra.Command{
		Use:   "send <uid>",
		Short: "Send a draft by UID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			uid, err := strconv.ParseUint(args[0], 10, 32)
			if err != nil {
				return fmt.Errorf("invalid uid: %s", args[0])
			}

			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if err := config.ValidateIMAP(cfg); err != nil {
				return err
			}
			if err := config.ValidateSMTP(cfg); err != nil {
				return err
			}

			service := imap.NewService()
			drafts := cfg.Defaults.DraftsMailbox
			if drafts == "" {
				drafts = "Drafts"
			}

			raw, err := service.FetchRawMessage(cfg, drafts, uint32(uid))
			if err != nil {
				return err
			}

			recipients, err := email.ExtractRecipients(raw)
			if err != nil {
				return err
			}
			if len(recipients) == 0 {
				return fmt.Errorf("draft has no recipients")
			}

			if err := smtp.Send(cfg, cfg.Auth.Username, recipients, raw); err != nil {
				return err
			}

			if !keep {
				if err := service.DeleteMessage(cfg, drafts, uint32(uid)); err != nil {
					return err
				}
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Draft sent.")
			return nil
		},
	}

	cmd.Flags().BoolVar(&keep, "keep", false, "Keep draft after sending")

	return cmd
}

package cli

import (
	"fmt"
	"strconv"
	"strings"

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
	var bodyHTML string
	var replyTo string
	var replyUID string
	var replyAll bool
	var quote bool
	var replyMailbox string
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

			if replyAll && strings.TrimSpace(replyUID) == "" {
				return fmt.Errorf("--reply-all requires --reply-uid")
			}
			if quote && strings.TrimSpace(replyUID) == "" {
				return fmt.Errorf("--quote requires --reply-uid")
			}

			var replyInfo *email.ReplyInfo
			var inReplyTo string
			var references string
			if strings.TrimSpace(replyUID) != "" {
				uid, err := strconv.ParseUint(replyUID, 10, 32)
				if err != nil {
					return fmt.Errorf("invalid reply uid: %s", replyUID)
				}
				if replyMailbox == "" {
					replyMailbox = "INBOX"
				}

				service := imap.NewService()
				raw, err := service.FetchRawMessage(cfg, replyMailbox, uint32(uid))
				if err != nil {
					return err
				}
				replyInfo, err = email.ExtractReplyInfo(raw, quote)
				if err != nil {
					return err
				}
				inReplyTo, references = email.BuildReplyHeaders(replyInfo)
				content, bodyHTML = email.ApplyQuoteToBodies(content, bodyHTML, quote, replyInfo)
				if strings.TrimSpace(subject) == "" && replyInfo.Subject != "" {
					subject = email.ReplySubject(replyInfo.Subject)
				}
			}

			var toList []string
			var ccList []string
			if replyInfo != nil {
				if replyAll {
					toList, ccList = email.BuildReplyAllRecipients(replyInfo, cfg.Auth.Username)
				} else {
					toList = email.BuildReplyRecipients(replyInfo, cfg.Auth.Username)
				}
				if strings.TrimSpace(to) != "" {
					toList = splitList(to)
				}
				if strings.TrimSpace(cc) != "" {
					ccList = splitList(cc)
				}
			} else {
				toList = splitList(to)
				ccList = splitList(cc)
			}

			bccList := splitList(bcc)

			msg, err := email.BuildMessage(email.ComposeInput{
				From:           cfg.Auth.Username,
				To:             toList,
				Cc:             ccList,
				Bcc:            bccList,
				ReplyTo:        replyTo,
				Subject:        subject,
				Body:           content,
				BodyHTML:       bodyHTML,
				InReplyTo:      inReplyTo,
				References:     references,
				Attachments:    attachments,
				StoreBccHeader: len(bccList) > 0,
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
	cmd.Flags().StringVar(&body, "body", "", "Message body (plain text)")
	cmd.Flags().StringVar(&bodyFile, "body-file", "", "Path to file containing message body ('-' for stdin)")
	cmd.Flags().StringVar(&bodyHTML, "body-html", "", "Message body (HTML)")
	cmd.Flags().StringVar(&replyTo, "reply-to", "", "Reply-To header address")
	cmd.Flags().StringVar(&replyUID, "reply-uid", "", "Reply to message UID (uses headers and thread)")
	cmd.Flags().BoolVar(&replyAll, "reply-all", false, "Reply-all using original recipients (requires --reply-uid)")
	cmd.Flags().BoolVar(&quote, "quote", false, "Include quoted original message (requires --reply-uid)")
	cmd.Flags().StringVar(&replyMailbox, "reply-mailbox", "INBOX", "Mailbox containing the reply target")
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

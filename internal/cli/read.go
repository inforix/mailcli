package cli

import (
	"fmt"
	"strconv"

	"mailcli/internal/config"
	"mailcli/internal/imap"

	"github.com/spf13/cobra"
)

func newReadCmd() *cobra.Command {
	var mailbox string

	cmd := &cobra.Command{
		Use:   "read <uid>",
		Short: "Read a message by UID",
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

			service := imap.NewService()
			detail, err := service.ReadMessage(cfg, mailbox, uint32(uid))
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "UID: %d\n", detail.UID)
			if detail.Subject != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Subject: %s\n", detail.Subject)
			}
			if detail.From != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "From: %s\n", detail.From)
			}
			if detail.To != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "To: %s\n", detail.To)
			}
			if detail.Cc != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Cc: %s\n", detail.Cc)
			}
			if !detail.Date.IsZero() {
				fmt.Fprintf(cmd.OutOrStdout(), "Date: %s\n", detail.Date.Format("2006-01-02 15:04:05 -0700"))
			}
			if len(detail.Attachments) > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "Attachments: %s\n", detail.Attachments)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "")
			fmt.Fprintln(cmd.OutOrStdout(), detail.TextBody)
			return nil
		},
	}

	cmd.Flags().StringVar(&mailbox, "mailbox", "INBOX", "Mailbox name")

	return cmd
}

package cli

import (
	"fmt"

	"mailcli/internal/config"
	"mailcli/internal/imap"

	"github.com/spf13/cobra"
)

func newMailCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mail",
		Short: "Mail operations",
	}
	cmd.AddCommand(newMailListCmd())
	return cmd
}

func newMailListCmd() *cobra.Command {
	var mailbox string
	var page int
	var pageSize int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List messages",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if err := config.ValidateIMAP(cfg); err != nil {
				return err
			}

			if mailbox == "" {
				mailbox = "INBOX"
			}

			service := imap.NewService()
			messages, total, err := service.ListMessages(cfg, mailbox, page, pageSize)
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Mailbox: %s (total %d)\n", mailbox, total)
			printMessages(cmd.OutOrStdout(), messages)
			return nil
		},
	}

	cmd.Flags().StringVar(&mailbox, "mailbox", "INBOX", "Mailbox name")
	cmd.Flags().IntVar(&page, "page", 1, "Page number (1-based, newest first)")
	cmd.Flags().IntVar(&pageSize, "page-size", 20, "Messages per page")

	return cmd
}

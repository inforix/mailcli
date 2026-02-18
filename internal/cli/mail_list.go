package cli

import (
	"errors"
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
	var threads bool

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
			if threads {
				threadSummaries, total, err := service.ListThreads(cfg, mailbox, page, pageSize)
				if err != nil {
					if !errors.Is(err, imap.ErrThreadUnsupported) {
						return err
					}
					fmt.Fprintln(cmd.ErrOrStderr(), "Server does not support THREAD; showing messages instead.")
				} else {
					fmt.Fprintf(cmd.OutOrStdout(), "Mailbox: %s (threads %d)\n", mailbox, total)
					printThreads(cmd.OutOrStdout(), threadSummaries)
					return nil
				}
			}

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
	cmd.Flags().BoolVar(&threads, "threads", false, "Show thread summaries when supported")

	return cmd
}

package cli

import (
	"errors"
	"fmt"

	"mailcli/internal/config"
	"mailcli/internal/imap"

	"github.com/spf13/cobra"
)

func newSearchCmd() *cobra.Command {
	var mailbox string
	var page int
	var pageSize int
	var threads bool

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search messages",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]

			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			if err := config.ValidateIMAP(cfg); err != nil {
				return err
			}

			service := imap.NewService()
			if threads {
				threadSummaries, total, err := service.SearchThreads(cfg, mailbox, query, page, pageSize)
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

			messages, total, err := service.SearchMessages(cfg, mailbox, query, page, pageSize)
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

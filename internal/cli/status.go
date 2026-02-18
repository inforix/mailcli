package cli

import (
	"fmt"

	"mailcli/internal/config"
	"mailcli/internal/imap"

	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show mailbox status",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if err := config.ValidateIMAP(cfg); err != nil {
				return err
			}

			service := imap.NewService()
			status, err := service.Status(cfg, "INBOX")
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "INBOX: %d messages, %d unseen\n", status.Messages, status.Unseen)
			return nil
		},
	}
	return cmd
}

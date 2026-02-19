package cli

import (
	"fmt"

	"mailcli/internal/config"
	"mailcli/internal/imap"

	"github.com/spf13/cobra"
)

func newMailboxesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mailboxes",
		Short: "Mailbox operations",
	}
	cmd.AddCommand(newMailboxesListCmd())
	cmd.AddCommand(newMailboxesCreateCmd())
	return cmd
}

func newMailboxesListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List mailboxes",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			if err := config.ValidateIMAP(cfg); err != nil {
				return err
			}

			service := imap.NewService()
			mailboxes, err := service.ListMailboxes(cfg)
			if err != nil {
				return err
			}

			for _, name := range mailboxes {
				fmt.Fprintln(cmd.OutOrStdout(), name)
			}
			return nil
		},
	}
	return cmd
}

func newMailboxesCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a mailbox",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			if err := config.ValidateIMAP(cfg); err != nil {
				return err
			}

			service := imap.NewService()
			if err := service.CreateMailbox(cfg, args[0]); err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Mailbox created.")
			return nil
		},
	}
	return cmd
}

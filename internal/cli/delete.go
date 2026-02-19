package cli

import (
	"fmt"
	"strconv"

	"mailcli/internal/config"
	"mailcli/internal/imap"

	"github.com/spf13/cobra"
)

func newDeleteCmd() *cobra.Command {
	var mailbox string

	cmd := &cobra.Command{
		Use:   "delete <uid>",
		Short: "Delete a message by UID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			uid, err := strconv.ParseUint(args[0], 10, 32)
			if err != nil {
				return fmt.Errorf("invalid uid: %s", args[0])
			}

			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			if err := config.ValidateIMAP(cfg); err != nil {
				return err
			}

			service := imap.NewService()
			if err := service.DeleteMessage(cfg, mailbox, uint32(uid)); err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Deleted.")
			return nil
		},
	}

	cmd.Flags().StringVar(&mailbox, "mailbox", "INBOX", "Mailbox name")

	return cmd
}

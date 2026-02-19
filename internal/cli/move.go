package cli

import (
	"fmt"
	"strconv"

	"mailcli/internal/config"
	"mailcli/internal/imap"

	"github.com/spf13/cobra"
)

func newMoveCmd() *cobra.Command {
	var mailbox string

	cmd := &cobra.Command{
		Use:   "move <uid> <mailbox>",
		Short: "Move a message to another mailbox",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			uid, err := strconv.ParseUint(args[0], 10, 32)
			if err != nil {
				return fmt.Errorf("invalid uid: %s", args[0])
			}
			dest := args[1]

			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			if err := config.ValidateIMAP(cfg); err != nil {
				return err
			}

			service := imap.NewService()
			if err := service.MoveMessage(cfg, mailbox, uint32(uid), dest); err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Moved.")
			return nil
		},
	}

	cmd.Flags().StringVar(&mailbox, "mailbox", "INBOX", "Source mailbox")

	return cmd
}

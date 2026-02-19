package cli

import (
	"fmt"
	"strconv"

	"mailcli/internal/config"
	"mailcli/internal/imap"

	"github.com/spf13/cobra"
)

func newTagCmd() *cobra.Command {
	var mailbox string

	cmd := &cobra.Command{
		Use:   "tag <uid> <tag>",
		Short: "Add a tag/label (keyword) to a message",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			uid, err := strconv.ParseUint(args[0], 10, 32)
			if err != nil {
				return fmt.Errorf("invalid uid: %s", args[0])
			}
			tag := args[1]

			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			if err := config.ValidateIMAP(cfg); err != nil {
				return err
			}

			service := imap.NewService()
			if err := service.AddTag(cfg, mailbox, uint32(uid), tag); err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Tagged.")
			return nil
		},
	}

	cmd.Flags().StringVar(&mailbox, "mailbox", "INBOX", "Mailbox name")

	return cmd
}

package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "mailcli",
		Short:        "mailcli is a CLI for IMAP/SMTP mail servers",
		SilenceUsage: true,
	}

	cmd.AddCommand(newAuthCmd())
	cmd.AddCommand(newStatusCmd())
	cmd.AddCommand(newInboxCmd())
	cmd.AddCommand(newMailCmd())
	cmd.AddCommand(newReadCmd())
	cmd.AddCommand(newSearchCmd())
	cmd.AddCommand(newSendCmd())
	cmd.AddCommand(newDraftCmd())
	cmd.AddCommand(newDeleteCmd())
	cmd.AddCommand(newMoveCmd())
	cmd.AddCommand(newTagCmd())
	cmd.AddCommand(newMailboxesCmd())
	cmd.AddCommand(newAttachmentsCmd())
	cmd.AddCommand(newConfigCmd())

	cmd.SetErr(os.Stderr)
	cmd.SetOut(os.Stdout)

	return cmd
}

func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

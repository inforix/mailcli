package cli

import "github.com/spf13/cobra"

func newInboxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inbox",
		Short: "Inbox operations",
	}
	cmd.AddCommand(newInboxListCmd())
	return cmd
}

func newInboxListCmd() *cobra.Command {
	listCmd := newMailListCmd()
	listCmd.Use = "list"
	listCmd.Short = "List messages in INBOX"
	listCmd.Args = cobra.NoArgs
	listCmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		return cmd.Flags().Set("mailbox", "INBOX")
	}
	return listCmd
}

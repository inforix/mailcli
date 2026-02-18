package cli

import (
	"fmt"
	"strconv"

	"mailcli/internal/config"
	"mailcli/internal/imap"

	"github.com/spf13/cobra"
)

func newAttachmentsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "attachments",
		Short: "Attachment operations",
	}
	cmd.AddCommand(newAttachmentsDownloadCmd())
	return cmd
}

func newAttachmentsDownloadCmd() *cobra.Command {
	var mailbox string
	var outputDir string

	cmd := &cobra.Command{
		Use:   "download <uid>",
		Short: "Download attachments from a message",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			uid, err := strconv.ParseUint(args[0], 10, 32)
			if err != nil {
				return fmt.Errorf("invalid uid: %s", args[0])
			}
			if outputDir == "" {
				outputDir = "."
			}

			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if err := config.ValidateIMAP(cfg); err != nil {
				return err
			}

			service := imap.NewService()
			files, err := service.DownloadAttachments(cfg, mailbox, uint32(uid), outputDir)
			if err != nil {
				return err
			}

			if len(files) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No attachments found.")
				return nil
			}
			for _, path := range files {
				fmt.Fprintln(cmd.OutOrStdout(), path)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&mailbox, "mailbox", "INBOX", "Mailbox name")
	cmd.Flags().StringVar(&outputDir, "output", ".", "Output directory")

	return cmd
}

package cli

import (
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"mailcli/internal/imap"
)

func printMessages(out io.Writer, messages []imap.MessageSummary) {
	tw := tabwriter.NewWriter(out, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "UID\tDATE\tFROM\tSUBJECT")
	for _, msg := range messages {
		date := ""
		if !msg.Date.IsZero() {
			date = msg.Date.Format(time.RFC3339)
		}
		fmt.Fprintf(tw, "%d\t%s\t%s\t%s\n", msg.UID, date, msg.From, msg.Subject)
	}
	_ = tw.Flush()
}

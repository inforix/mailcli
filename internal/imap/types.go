package imap

import "time"

type MessageSummary struct {
	UID     uint32
	Subject string
	From    string
	Date    time.Time
	Size    uint32
	Flags   []string
}

type MessageDetail struct {
	UID         uint32
	Subject     string
	From        string
	To          string
	Cc          string
	Date        time.Time
	TextBody    string
	Attachments []string
}

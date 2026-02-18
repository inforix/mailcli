package email

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/emersion/go-message/mail"
)

type ComposeInput struct {
	From           string
	To             []string
	Cc             []string
	Bcc            []string
	Subject        string
	Body           string
	Attachments    []string
	StoreBccHeader bool
}

func BuildMessage(in ComposeInput) ([]byte, error) {
	if in.From == "" {
		return nil, fmt.Errorf("from address is required")
	}

	var buf bytes.Buffer

	writeHeader(&buf, "From", in.From)
	if len(in.To) > 0 {
		writeHeader(&buf, "To", strings.Join(in.To, ", "))
	}
	if len(in.Cc) > 0 {
		writeHeader(&buf, "Cc", strings.Join(in.Cc, ", "))
	}
	if in.Subject != "" {
		writeHeader(&buf, "Subject", in.Subject)
	}
	if in.StoreBccHeader && len(in.Bcc) > 0 {
		writeHeader(&buf, "X-Mailcli-Bcc", strings.Join(in.Bcc, ", "))
	}
	writeHeader(&buf, "Date", time.Now().Format(time.RFC1123Z))
	writeHeader(&buf, "MIME-Version", "1.0")

	if len(in.Attachments) == 0 {
		writeHeader(&buf, "Content-Type", "text/plain; charset=\"utf-8\"")
		writeHeader(&buf, "Content-Transfer-Encoding", "quoted-printable")
		buf.WriteString("\r\n")
		qp := quotedprintable.NewWriter(&buf)
		if _, err := qp.Write([]byte(in.Body)); err != nil {
			return nil, err
		}
		if err := qp.Close(); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}

	writer := multipart.NewWriter(&buf)
	boundary := writer.Boundary()
	writeHeader(&buf, "Content-Type", fmt.Sprintf("multipart/mixed; boundary=%q", boundary))
	buf.WriteString("\r\n")

	textHeader := textproto.MIMEHeader{}
	textHeader.Set("Content-Type", "text/plain; charset=\"utf-8\"")
	textHeader.Set("Content-Transfer-Encoding", "quoted-printable")
	textPart, err := writer.CreatePart(textHeader)
	if err != nil {
		return nil, err
	}
	qp := quotedprintable.NewWriter(textPart)
	if _, err := qp.Write([]byte(in.Body)); err != nil {
		return nil, err
	}
	if err := qp.Close(); err != nil {
		return nil, err
	}

	for _, attachmentPath := range in.Attachments {
		if attachmentPath == "" {
			continue
		}
		data, err := os.ReadFile(attachmentPath)
		if err != nil {
			return nil, err
		}
		filename := filepath.Base(attachmentPath)
		contentType := mime.TypeByExtension(filepath.Ext(filename))
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		partHeader := textproto.MIMEHeader{}
		partHeader.Set("Content-Type", fmt.Sprintf("%s; name=\"%s\"", contentType, filename))
		partHeader.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
		partHeader.Set("Content-Transfer-Encoding", "base64")

		part, err := writer.CreatePart(partHeader)
		if err != nil {
			return nil, err
		}
		if err := writeBase64(part, data); err != nil {
			return nil, err
		}
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func ExtractRecipients(raw []byte) ([]string, error) {
	reader, err := mail.CreateReader(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}

	header := reader.Header
	recipients := []string{}

	addFromHeader := func(field string) error {
		list, err := header.AddressList(field)
		if err != nil {
			if err == mail.ErrHeaderNotPresent {
				return nil
			}
			return err
		}
		for _, addr := range list {
			recipients = append(recipients, addr.Address)
		}
		return nil
	}

	if err := addFromHeader("To"); err != nil {
		return nil, err
	}
	if err := addFromHeader("Cc"); err != nil {
		return nil, err
	}
	if err := addFromHeader("Bcc"); err != nil {
		return nil, err
	}

	if bcc := header.Get("X-Mailcli-Bcc"); bcc != "" {
		extra := strings.Split(bcc, ",")
		for _, part := range extra {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				recipients = append(recipients, trimmed)
			}
		}
	}

	return recipients, nil
}

func writeHeader(buf *bytes.Buffer, key, value string) {
	if value == "" {
		return
	}
	buf.WriteString(key)
	buf.WriteString(": ")
	buf.WriteString(value)
	buf.WriteString("\r\n")
}

func writeBase64(w io.Writer, data []byte) error {
	encoded := base64.StdEncoding.EncodeToString(data)
	for len(encoded) > 76 {
		if _, err := w.Write([]byte(encoded[:76] + "\r\n")); err != nil {
			return err
		}
		encoded = encoded[76:]
	}
	if len(encoded) > 0 {
		if _, err := w.Write([]byte(encoded + "\r\n")); err != nil {
			return err
		}
	}
	return nil
}

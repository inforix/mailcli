package imap

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"mailcli/internal/config"

	"github.com/emersion/go-imap"
	imapclient "github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
)

type Client interface {
	Login(username, password string) error
	Logout() error
	StartTLS(config *tls.Config) error
	Select(name string, readOnly bool) (*imap.MailboxStatus, error)
	Status(name string, items []imap.StatusItem) (*imap.MailboxStatus, error)
	List(ref, name string, ch chan *imap.MailboxInfo) error
	Create(name string) error
	UidSearch(criteria *imap.SearchCriteria) ([]uint32, error)
	UidFetch(seqset *imap.SeqSet, items []imap.FetchItem, ch chan *imap.Message) error
	UidStore(seqset *imap.SeqSet, item imap.StoreItem, flags []interface{}) error
	UidMove(seqset *imap.SeqSet, mailbox string) error
	UidCopy(seqset *imap.SeqSet, mailbox string) error
	Append(mailbox string, flags []string, date time.Time, msg imap.Literal) error
	Expunge(ch chan uint32) error
}

type Service struct {
	Connector func(cfg config.Config) (Client, error)
}

func NewService() *Service {
	return &Service{Connector: Connect}
}

func Connect(cfg config.Config) (Client, error) {
	addr := fmt.Sprintf("%s:%d", cfg.IMAP.Host, cfg.IMAP.Port)
	var c *imapclient.Client
	var err error

	if cfg.IMAP.TLS {
		tlsConfig := &tls.Config{
			ServerName:         cfg.IMAP.Host,
			InsecureSkipVerify: cfg.IMAP.InsecureSkipVerify,
		}
		c, err = imapclient.DialTLS(addr, tlsConfig)
	} else {
		c, err = imapclient.Dial(addr)
		if err == nil && cfg.IMAP.StartTLS {
			tlsConfig := &tls.Config{
				ServerName:         cfg.IMAP.Host,
				InsecureSkipVerify: cfg.IMAP.InsecureSkipVerify,
			}
			if err := c.StartTLS(tlsConfig); err != nil {
				_ = c.Logout()
				return nil, err
			}
		}
	}
	if err != nil {
		return nil, err
	}

	if err := c.Login(cfg.Auth.Username, cfg.Auth.Password); err != nil {
		_ = c.Logout()
		return nil, err
	}

	return c, nil
}

func (s *Service) withClient(cfg config.Config, fn func(Client) error) error {
	connector := s.Connector
	if connector == nil {
		connector = Connect
	}
	client, err := connector(cfg)
	if err != nil {
		return err
	}
	defer func() {
		_ = client.Logout()
	}()
	return fn(client)
}

func (s *Service) Status(cfg config.Config, mailbox string) (*imap.MailboxStatus, error) {
	var status *imap.MailboxStatus
	err := s.withClient(cfg, func(c Client) error {
		mb, err := c.Status(mailbox, []imap.StatusItem{imap.StatusMessages, imap.StatusUnseen})
		if err != nil {
			return err
		}
		status = mb
		return nil
	})
	return status, err
}

func (s *Service) ListMailboxes(cfg config.Config) ([]string, error) {
	mailboxes := []string{}
	err := s.withClient(cfg, func(c Client) error {
		ch := make(chan *imap.MailboxInfo, 10)
		done := make(chan error, 1)
		go func() {
			done <- c.List("", "*", ch)
		}()
		for mbox := range ch {
			mailboxes = append(mailboxes, mbox.Name)
		}
		return <-done
	})
	return mailboxes, err
}

func (s *Service) CreateMailbox(cfg config.Config, name string) error {
	return s.withClient(cfg, func(c Client) error {
		return c.Create(name)
	})
}

func (s *Service) ListMessages(cfg config.Config, mailbox string, page, pageSize int) ([]MessageSummary, int, error) {
	return s.listMessagesWithCriteria(cfg, mailbox, nil, page, pageSize)
}

func (s *Service) SearchMessages(cfg config.Config, mailbox, query string, page, pageSize int) ([]MessageSummary, int, error) {
	criteria := imap.NewSearchCriteria()
	criteria.Text = []string{query}
	return s.listMessagesWithCriteria(cfg, mailbox, criteria, page, pageSize)
}

func (s *Service) listMessagesWithCriteria(cfg config.Config, mailbox string, criteria *imap.SearchCriteria, page, pageSize int) ([]MessageSummary, int, error) {
	var messages []MessageSummary
	var total int

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	err := s.withClient(cfg, func(c Client) error {
		if _, err := c.Select(mailbox, true); err != nil {
			return err
		}

		if criteria == nil {
			criteria = imap.NewSearchCriteria()
		}

		uids, err := c.UidSearch(criteria)
		if err != nil {
			return err
		}
		sort.Slice(uids, func(i, j int) bool { return uids[i] < uids[j] })
		total = len(uids)
		if total == 0 {
			return nil
		}

		end := total - (page-1)*pageSize
		if end <= 0 {
			return nil
		}
		start := end - pageSize
		if start < 0 {
			start = 0
		}
		subset := uids[start:end]
		if len(subset) == 0 {
			return nil
		}

		seqset := new(imap.SeqSet)
		seqset.AddNum(subset...)

		items := []imap.FetchItem{imap.FetchEnvelope, imap.FetchFlags, imap.FetchUid, imap.FetchRFC822Size}
		ch := make(chan *imap.Message, len(subset))
		done := make(chan error, 1)
		go func() {
			done <- c.UidFetch(seqset, items, ch)
		}()
		for msg := range ch {
			if msg == nil || msg.Envelope == nil {
				continue
			}
			messages = append(messages, MessageSummary{
				UID:     msg.Uid,
				Subject: msg.Envelope.Subject,
				From:    formatIMAPAddresses(msg.Envelope.From),
				Date:    msg.Envelope.Date,
				Size:    msg.Size,
				Flags:   msg.Flags,
			})
		}
		if err := <-done; err != nil {
			return err
		}
		return nil
	})

	sort.Slice(messages, func(i, j int) bool { return messages[i].UID > messages[j].UID })

	return messages, total, err
}

func (s *Service) ReadMessage(cfg config.Config, mailbox string, uid uint32) (MessageDetail, error) {
	detail := MessageDetail{}
	err := s.withClient(cfg, func(c Client) error {
		if _, err := c.Select(mailbox, true); err != nil {
			return err
		}

		seqset := new(imap.SeqSet)
		seqset.AddNum(uid)
		section := &imap.BodySectionName{}
		items := []imap.FetchItem{imap.FetchUid, imap.FetchEnvelope, section.FetchItem()}
		ch := make(chan *imap.Message, 1)
		done := make(chan error, 1)
		go func() {
			done <- c.UidFetch(seqset, items, ch)
		}()
		msg := <-ch
		if msg == nil {
			return fmt.Errorf("message %d not found", uid)
		}
		if err := <-done; err != nil {
			return err
		}

		detail.UID = msg.Uid
		if msg.Envelope != nil {
			detail.Subject = msg.Envelope.Subject
			detail.From = formatIMAPAddresses(msg.Envelope.From)
			detail.To = formatIMAPAddresses(msg.Envelope.To)
			detail.Cc = formatIMAPAddresses(msg.Envelope.Cc)
			detail.Date = msg.Envelope.Date
		}

		body := msg.GetBody(section)
		if body == nil {
			return fmt.Errorf("message body not available")
		}
		reader, err := mail.CreateReader(body)
		if err != nil {
			return err
		}

		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			switch header := part.Header.(type) {
			case *mail.InlineHeader:
				contentType, _, _ := header.ContentType()
				if strings.HasPrefix(contentType, "text/plain") && detail.TextBody == "" {
					data, err := io.ReadAll(part.Body)
					if err != nil {
						return err
					}
					detail.TextBody = string(data)
				}
			case *mail.AttachmentHeader:
				filename, err := header.Filename()
				if err != nil {
					continue
				}
				detail.Attachments = append(detail.Attachments, filename)
			}
		}

		return nil
	})

	return detail, err
}

func (s *Service) FetchRawMessage(cfg config.Config, mailbox string, uid uint32) ([]byte, error) {
	var raw []byte
	err := s.withClient(cfg, func(c Client) error {
		if _, err := c.Select(mailbox, true); err != nil {
			return err
		}

		seqset := new(imap.SeqSet)
		seqset.AddNum(uid)
		section := &imap.BodySectionName{}
		items := []imap.FetchItem{section.FetchItem()}
		ch := make(chan *imap.Message, 1)
		done := make(chan error, 1)
		go func() {
			done <- c.UidFetch(seqset, items, ch)
		}()
		msg := <-ch
		if msg == nil {
			return fmt.Errorf("message %d not found", uid)
		}
		if err := <-done; err != nil {
			return err
		}
		body := msg.GetBody(section)
		if body == nil {
			return fmt.Errorf("message body not available")
		}
		data, err := io.ReadAll(body)
		if err != nil {
			return err
		}
		raw = data
		return nil
	})

	return raw, err
}

func (s *Service) DeleteMessage(cfg config.Config, mailbox string, uid uint32) error {
	return s.withClient(cfg, func(c Client) error {
		if _, err := c.Select(mailbox, false); err != nil {
			return err
		}
		seqset := new(imap.SeqSet)
		seqset.AddNum(uid)
		item := imap.FormatFlagsOp(imap.AddFlags, true)
		if err := c.UidStore(seqset, item, []interface{}{imap.DeletedFlag}); err != nil {
			return err
		}
		expunge := make(chan uint32)
		done := make(chan error, 1)
		go func() {
			done <- c.Expunge(expunge)
		}()
		for range expunge {
		}
		return <-done
	})
}

func (s *Service) MoveMessage(cfg config.Config, mailbox string, uid uint32, dest string) error {
	return s.withClient(cfg, func(c Client) error {
		if _, err := c.Select(mailbox, false); err != nil {
			return err
		}
		seqset := new(imap.SeqSet)
		seqset.AddNum(uid)
		if err := c.UidMove(seqset, dest); err == nil {
			return nil
		}
		if err := c.UidCopy(seqset, dest); err != nil {
			return err
		}
		item := imap.FormatFlagsOp(imap.AddFlags, true)
		if err := c.UidStore(seqset, item, []interface{}{imap.DeletedFlag}); err != nil {
			return err
		}
		expunge := make(chan uint32)
		done := make(chan error, 1)
		go func() {
			done <- c.Expunge(expunge)
		}()
		for range expunge {
		}
		return <-done
	})
}

func (s *Service) AddTag(cfg config.Config, mailbox string, uid uint32, tag string) error {
	return s.withClient(cfg, func(c Client) error {
		if _, err := c.Select(mailbox, false); err != nil {
			return err
		}
		seqset := new(imap.SeqSet)
		seqset.AddNum(uid)
		item := imap.FormatFlagsOp(imap.AddFlags, true)
		return c.UidStore(seqset, item, []interface{}{tag})
	})
}

func (s *Service) SaveDraft(cfg config.Config, mailbox string, raw []byte) error {
	return s.withClient(cfg, func(c Client) error {
		return c.Append(mailbox, []string{}, time.Now(), bytes.NewReader(raw))
	})
}

func (s *Service) DownloadAttachments(cfg config.Config, mailbox string, uid uint32, dir string) ([]string, error) {
	raw, err := s.FetchRawMessage(cfg, mailbox, uid)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	reader, err := mail.CreateReader(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}

	saved := []string{}
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		header, ok := part.Header.(*mail.AttachmentHeader)
		if !ok {
			continue
		}
		filename, err := header.Filename()
		if err != nil || filename == "" {
			filename = fmt.Sprintf("attachment-%d", len(saved)+1)
		}
		filename = filepath.Base(filename)
		target := filepath.Join(dir, filename)
		target = ensureUniqueFilename(target)
		file, err := os.Create(target)
		if err != nil {
			return nil, err
		}
		if _, err := io.Copy(file, part.Body); err != nil {
			_ = file.Close()
			return nil, err
		}
		if err := file.Close(); err != nil {
			return nil, err
		}
		saved = append(saved, target)
	}

	return saved, nil
}

func formatIMAPAddresses(addrs []*imap.Address) string {
	if len(addrs) == 0 {
		return ""
	}
	parts := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		if addr == nil {
			continue
		}
		mailbox := addr.MailboxName
		host := addr.HostName
		full := mailbox
		if host != "" {
			full = mailbox + "@" + host
		}
		if addr.PersonalName != "" {
			parts = append(parts, fmt.Sprintf("%s <%s>", addr.PersonalName, full))
		} else {
			parts = append(parts, full)
		}
	}
	return strings.Join(parts, ", ")
}

func ensureUniqueFilename(path string) string {
	if _, err := os.Stat(path); err != nil {
		return path
	}
	base := strings.TrimSuffix(path, filepath.Ext(path))
	ext := filepath.Ext(path)
	for i := 1; i < 1000; i++ {
		candidate := fmt.Sprintf("%s-%d%s", base, i, ext)
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
	return path
}

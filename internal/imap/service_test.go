package imap

import (
	"crypto/tls"
	"testing"
	"time"

	"mailcli/internal/config"

	"github.com/emersion/go-imap"
)

type mockClient struct {
	listNames []string
	loggedOut bool
}

func (m *mockClient) Login(username, password string) error { return nil }
func (m *mockClient) Logout() error {
	m.loggedOut = true
	return nil
}
func (m *mockClient) StartTLS(config *tls.Config) error { return nil }
func (m *mockClient) Select(name string, readOnly bool) (*imap.MailboxStatus, error) {
	return &imap.MailboxStatus{Name: name}, nil
}
func (m *mockClient) Status(name string, items []imap.StatusItem) (*imap.MailboxStatus, error) {
	return &imap.MailboxStatus{Name: name}, nil
}
func (m *mockClient) List(ref, name string, ch chan *imap.MailboxInfo) error {
	for _, mailbox := range m.listNames {
		ch <- &imap.MailboxInfo{Name: mailbox}
	}
	close(ch)
	return nil
}
func (m *mockClient) Create(name string) error { return nil }
func (m *mockClient) UidSearch(criteria *imap.SearchCriteria) ([]uint32, error) {
	return nil, nil
}
func (m *mockClient) UidFetch(seqset *imap.SeqSet, items []imap.FetchItem, ch chan *imap.Message) error {
	close(ch)
	return nil
}
func (m *mockClient) UidStore(seqset *imap.SeqSet, item imap.StoreItem, flags []interface{}) error {
	return nil
}
func (m *mockClient) UidMove(seqset *imap.SeqSet, mailbox string) error { return nil }
func (m *mockClient) UidCopy(seqset *imap.SeqSet, mailbox string) error { return nil }
func (m *mockClient) Append(mailbox string, flags []string, date time.Time, msg imap.Literal) error {
	return nil
}
func (m *mockClient) Expunge(ch chan uint32) error {
	if ch != nil {
		close(ch)
	}
	return nil
}

func TestListMailboxesWithMock(t *testing.T) {
	mock := &mockClient{listNames: []string{"INBOX", "Archive"}}
	svc := &Service{Connector: func(cfg config.Config) (Client, error) {
		return mock, nil
	}}

	mailboxes, err := svc.ListMailboxes(config.Config{})
	if err != nil {
		t.Fatalf("list mailboxes: %v", err)
	}
	if len(mailboxes) != 2 {
		t.Fatalf("expected 2 mailboxes, got %d", len(mailboxes))
	}
	if mailboxes[0] != "INBOX" || mailboxes[1] != "Archive" {
		t.Fatalf("unexpected mailboxes: %v", mailboxes)
	}
	if !mock.loggedOut {
		t.Fatalf("expected logout to be called")
	}
}

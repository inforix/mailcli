# mailcli

A minimal, cobra-based IMAP/SMTP CLI inspired by `gogcli`, designed for generic mail servers (not Gmail-specific).

## Build

```bash
go build -o mailcli ./cmd/mailcli
```

## Config

Config is stored at `~/.config/mailcli/config.yaml`.

Example:

```yaml
imap:
  host: imap.example.com
  port: 993
  tls: true
  starttls: false
  insecure_skip_verify: false
smtp:
  host: smtp.example.com
  port: 587
  tls: false
  starttls: true
  insecure_skip_verify: false
auth:
  username: you@example.com
  password: app-password
defaults:
  drafts_mailbox: Drafts
```

Environment variable overrides are supported with the `MAILCLI_` prefix. Nested keys use underscores:

```
MAILCLI_IMAP_HOST=imap.example.com
MAILCLI_SMTP_HOST=smtp.example.com
MAILCLI_AUTH_USERNAME=you@example.com
MAILCLI_AUTH_PASSWORD=app-password
```

## Quick Setup

```bash
./mailcli auth login \
  --imap-host imap.example.com --imap-port 993 --imap-tls \
  --smtp-host smtp.example.com --smtp-port 587 --smtp-starttls \
  --username you@example.com --password app-password
```

## Usage Examples

```bash
./mailcli status
./mailcli inbox list --page 1 --page-size 20
./mailcli inbox list --threads
./mailcli mail list --mailbox Archive
./mailcli search "invoice" --mailbox INBOX
./mailcli read 12345
./mailcli read 12345 --html

./mailcli send \
  --to "alice@example.com,bob@example.com" \
  --cc "team@example.com" \
  --subject "Weekly update" \
  --body "Hello team..." \
  --attachment ./report.pdf

./mailcli draft save --to "alice@example.com" --subject "Draft" --body "Work in progress"
./mailcli draft list
./mailcli draft send 42

./mailcli delete 12345
./mailcli move 12345 Archive
./mailcli tag 12345 FollowUp

./mailcli mailboxes list
./mailcli mailboxes create "Project X"

./mailcli attachments download 12345 --output ./attachments

./mailcli config show
./mailcli config edit
```

## Notes

- `read`, `list`, `search`, and other IMAP operations use message UIDs.
- Draft BCC recipients are stored in an `X-Mailcli-Bcc` header so they can be used when sending drafts.

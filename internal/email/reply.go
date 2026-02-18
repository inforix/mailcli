package email

import (
	"bytes"
	"fmt"
	"html"
	"io"
	"net/mail"
	"regexp"
	"strings"

	gomail "github.com/emersion/go-message/mail"
)

type ReplyInfo struct {
	MessageID  string
	References string
	From       string
	ReplyTo    string
	To         []string
	Cc         []string
	Date       string
	Subject    string
	Body       string
	BodyHTML   string
}

func ExtractReplyInfo(raw []byte, includeBodies bool) (*ReplyInfo, error) {
	r, err := gomail.CreateReader(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}

	header := r.Header
	info := &ReplyInfo{
		MessageID:  firstHeaderValue(header, "Message-ID", "Message-Id"),
		References: strings.TrimSpace(header.Get("References")),
		From:       header.Get("From"),
		ReplyTo:    header.Get("Reply-To"),
		To:         parseEmailAddresses(header.Get("To")),
		Cc:         parseEmailAddresses(header.Get("Cc")),
		Date:       header.Get("Date"),
		Subject:    header.Get("Subject"),
	}

	if !includeBodies {
		return info, nil
	}

	for {
		part, err := r.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
		switch header := part.Header.(type) {
		case *gomail.InlineHeader:
			contentType, _, _ := header.ContentType()
			if strings.HasPrefix(contentType, "text/plain") && info.Body == "" {
				info.Body = readAll(part.Body)
			}
			if strings.HasPrefix(contentType, "text/html") && info.BodyHTML == "" {
				info.BodyHTML = readAll(part.Body)
			}
		}
	}

	if info.Body != "" && looksLikeHTML(info.Body) {
		info.Body = ""
	}

	return info, nil
}

func BuildReplyHeaders(info *ReplyInfo) (string, string) {
	if info == nil {
		return "", ""
	}
	messageID := strings.TrimSpace(info.MessageID)
	inReplyTo := messageID
	refs := strings.TrimSpace(info.References)
	if refs == "" {
		refs = messageID
	} else if messageID != "" && !strings.Contains(refs, messageID) {
		refs = refs + " " + messageID
	}
	return inReplyTo, refs
}

func BuildReplyRecipients(info *ReplyInfo, selfEmail string) []string {
	if info == nil {
		return nil
	}
	replyAddress := strings.TrimSpace(info.ReplyTo)
	if replyAddress == "" {
		replyAddress = info.From
	}
	toAddrs := parseEmailAddresses(replyAddress)
	toAddrs = filterOutSelf(toAddrs, selfEmail)
	return deduplicateAddresses(toAddrs)
}

func BuildReplyAllRecipients(info *ReplyInfo, selfEmail string) (to, cc []string) {
	if info == nil {
		return nil, nil
	}
	replyAddress := strings.TrimSpace(info.ReplyTo)
	if replyAddress == "" {
		replyAddress = info.From
	}
	toAddrs := parseEmailAddresses(replyAddress)
	toAddrs = append(toAddrs, info.To...)
	toAddrs = filterOutSelf(toAddrs, selfEmail)
	toAddrs = deduplicateAddresses(toAddrs)

	ccAddrs := filterOutSelf(info.Cc, selfEmail)
	ccAddrs = deduplicateAddresses(ccAddrs)

	toSet := make(map[string]bool)
	for _, addr := range toAddrs {
		toSet[strings.ToLower(addr)] = true
	}
	filteredCc := make([]string, 0, len(ccAddrs))
	for _, addr := range ccAddrs {
		if !toSet[strings.ToLower(addr)] {
			filteredCc = append(filteredCc, addr)
		}
	}

	return toAddrs, filteredCc
}

func ReplySubject(original string) string {
	trimmed := strings.TrimSpace(original)
	if trimmed == "" {
		return ""
	}
	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "re:") {
		return trimmed
	}
	return "Re: " + trimmed
}

func ApplyQuoteToBodies(plainBody string, htmlBody string, quote bool, info *ReplyInfo) (string, string) {
	if !quote || info == nil {
		return plainBody, htmlBody
	}
	if info.Body == "" && info.BodyHTML == "" {
		return plainBody, htmlBody
	}

	userPlain := plainBody
	outPlain := plainBody
	if info.Body != "" {
		outPlain += formatQuotedMessage(info.From, info.Date, info.Body)
	}

	quoteContent := info.BodyHTML
	if quoteContent == "" && info.Body != "" {
		quoteContent = escapeTextToHTML(info.Body)
	}
	if quoteContent == "" {
		return outPlain, htmlBody
	}

	quoteHTML := formatQuotedMessageHTMLWithContent(info.From, info.Date, quoteContent)

	outHTML := htmlBody
	if strings.TrimSpace(outHTML) == "" {
		outHTML = escapeTextToHTML(strings.TrimSpace(userPlain)) + quoteHTML
	} else {
		outHTML += quoteHTML
	}

	return outPlain, outHTML
}

func escapeTextToHTML(value string) string {
	value = html.EscapeString(value)
	return strings.ReplaceAll(value, "\n", "<br>\n")
}

func formatQuotedMessage(from, date, body string) string {
	if body == "" {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n\n")

	switch {
	case date != "" && from != "":
		sb.WriteString(fmt.Sprintf("On %s, %s wrote:\n", date, from))
	case from != "":
		sb.WriteString(fmt.Sprintf("%s wrote:\n", from))
	default:
		sb.WriteString("Original message:\n")
	}

	lines := strings.Split(body, "\n")
	for _, line := range lines {
		sb.WriteString("> ")
		sb.WriteString(line)
		sb.WriteString("\n")
	}

	return sb.String()
}

func formatQuotedMessageHTMLWithContent(from, date, htmlContent string) string {
	senderName := from
	if addr, err := mail.ParseAddress(from); err == nil && addr.Name != "" {
		senderName = addr.Name
	}

	dateStr := date
	if dateStr == "" {
		dateStr = "an earlier date"
	}

	return fmt.Sprintf(`<br><br><div class="gmail_quote"><div class="gmail_attr">On %s, %s wrote:</div><blockquote class="gmail_quote" style="margin:0 0 0 .8ex;border-left:1px #ccc solid;padding-left:1ex">%s</blockquote></div>`,
		html.EscapeString(dateStr),
		html.EscapeString(senderName),
		htmlContent)
}

func parseEmailAddresses(header string) []string {
	header = strings.TrimSpace(header)
	if header == "" {
		return nil
	}
	addrs, err := mail.ParseAddressList(header)
	if err != nil {
		return parseEmailAddressesFallback(header)
	}
	result := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		if addr.Address != "" {
			result = append(result, strings.ToLower(addr.Address))
		}
	}
	return result
}

func parseEmailAddressesFallback(header string) []string {
	parts := strings.Split(header, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if start := strings.LastIndex(p, "<"); start != -1 {
			if end := strings.LastIndex(p, ">"); end > start {
				email := strings.TrimSpace(p[start+1 : end])
				if email != "" {
					result = append(result, strings.ToLower(email))
				}
				continue
			}
		}
		if strings.Contains(p, "@") {
			result = append(result, strings.ToLower(p))
		}
	}
	return result
}

func filterOutSelf(addresses []string, selfEmail string) []string {
	selfLower := strings.ToLower(selfEmail)
	result := make([]string, 0, len(addresses))
	for _, addr := range addresses {
		if strings.ToLower(addr) != selfLower {
			result = append(result, addr)
		}
	}
	return result
}

func deduplicateAddresses(addresses []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(addresses))
	for _, addr := range addresses {
		lower := strings.ToLower(addr)
		if !seen[lower] {
			seen[lower] = true
			result = append(result, addr)
		}
	}
	return result
}

func looksLikeHTML(value string) bool {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return false
	}
	return strings.HasPrefix(trimmed, "<!doctype") ||
		strings.HasPrefix(trimmed, "<html") ||
		strings.HasPrefix(trimmed, "<head") ||
		strings.HasPrefix(trimmed, "<body") ||
		strings.HasPrefix(trimmed, "<meta") ||
		strings.Contains(trimmed, "<html")
}

var (
	scriptPattern     = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	stylePattern      = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	htmlTagPattern    = regexp.MustCompile(`<[^>]*>`)
	whitespacePattern = regexp.MustCompile(`\s+`)
)

func StripHTMLTags(s string) string {
	s = scriptPattern.ReplaceAllString(s, "")
	s = stylePattern.ReplaceAllString(s, "")
	s = htmlTagPattern.ReplaceAllString(s, " ")
	s = whitespacePattern.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

func firstHeaderValue(header gomail.Header, names ...string) string {
	for _, name := range names {
		if value := strings.TrimSpace(header.Get(name)); value != "" {
			return value
		}
	}
	return ""
}

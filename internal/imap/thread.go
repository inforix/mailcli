package imap

import (
	"errors"
	"sort"
	"strings"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/commands"
	"github.com/emersion/go-imap/responses"
)

var ErrThreadUnsupported = errors.New("imap server does not support THREAD")

type threadClient interface {
	Execute(cmdr imap.Commander, h responses.Handler) (*imap.StatusResp, error)
	Capability() (map[string]bool, error)
}

type threadCommand struct {
	Algorithm string
	Charset   string
	Criteria  *imap.SearchCriteria
}

func (cmd *threadCommand) Command() *imap.Command {
	criteria := cmd.Criteria
	if criteria == nil {
		criteria = imap.NewSearchCriteria()
	}
	charset := cmd.Charset
	if charset == "" {
		charset = "UTF-8"
	}
	args := []interface{}{
		imap.RawString(strings.ToUpper(cmd.Algorithm)),
		imap.RawString(charset),
	}
	args = append(args, criteria.Format()...)
	return &imap.Command{
		Name:      "THREAD",
		Arguments: args,
	}
}

type threadResponse struct {
	Threads [][]uint32
}

func (r *threadResponse) Handle(resp imap.Resp) error {
	name, fields, ok := imap.ParseNamedResp(resp)
	if !ok || name != "THREAD" {
		return responses.ErrUnhandled
	}
	threads, err := parseThreadFields(fields)
	if err != nil {
		return err
	}
	r.Threads = threads
	return nil
}

func parseThreadFields(fields []interface{}) ([][]uint32, error) {
	if len(fields) == 0 {
		return nil, nil
	}
	threads := make([][]uint32, 0, len(fields))
	for _, field := range fields {
		uids := []uint32{}
		if err := collectThreadUIDs(field, &uids); err != nil {
			return nil, err
		}
		uids = dedupeThreadUIDs(uids)
		if len(uids) > 0 {
			threads = append(threads, uids)
		}
	}
	return threads, nil
}

func collectThreadUIDs(field interface{}, out *[]uint32) error {
	switch value := field.(type) {
	case []interface{}:
		for _, item := range value {
			if err := collectThreadUIDs(item, out); err != nil {
				return err
			}
		}
		return nil
	case nil:
		return nil
	default:
		uid, err := imap.ParseNumber(value)
		if err != nil {
			return err
		}
		*out = append(*out, uid)
		return nil
	}
}

func dedupeThreadUIDs(uids []uint32) []uint32 {
	if len(uids) < 2 {
		return uids
	}
	seen := make(map[uint32]bool, len(uids))
	out := make([]uint32, 0, len(uids))
	for _, uid := range uids {
		if seen[uid] {
			continue
		}
		seen[uid] = true
		out = append(out, uid)
	}
	return out
}

func selectThreadAlgorithm(caps map[string]bool) (string, bool) {
	if len(caps) == 0 {
		return "", false
	}
	algorithms := []string{}
	for cap := range caps {
		upper := strings.ToUpper(cap)
		if strings.HasPrefix(upper, "THREAD=") {
			algorithms = append(algorithms, strings.TrimPrefix(upper, "THREAD="))
		}
	}
	if len(algorithms) == 0 {
		return "", false
	}
	for _, preferred := range []string{"REFERENCES", "REFS", "ORDEREDSUBJECT", "ORDERED-SUBJECT"} {
		for _, alg := range algorithms {
			if alg == preferred {
				return alg, true
			}
		}
	}
	sort.Strings(algorithms)
	return algorithms[0], true
}

func executeThread(tc threadClient, algorithm, charset string, criteria *imap.SearchCriteria) ([][]uint32, *imap.StatusResp, error) {
	cmd := &threadCommand{
		Algorithm: algorithm,
		Charset:   charset,
		Criteria:  criteria,
	}
	uidCmd := &commands.Uid{Cmd: cmd}
	res := &threadResponse{}
	status, err := tc.Execute(uidCmd, res)
	if err != nil {
		return nil, status, err
	}
	if statusErr := status.Err(); statusErr != nil {
		return nil, status, statusErr
	}
	return res.Threads, status, nil
}

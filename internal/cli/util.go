package cli

import (
	"fmt"
	"os"
	"strings"
)

func splitList(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func loadBody(body, bodyFile string) (string, error) {
	if bodyFile == "" {
		return body, nil
	}
	data, err := os.ReadFile(bodyFile)
	if err != nil {
		return "", err
	}
	if body != "" {
		return "", fmt.Errorf("use either --body or --body-file")
	}
	return string(data), nil
}

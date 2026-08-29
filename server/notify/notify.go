// Package notify sends push notifications via ntfy (https://ntfy.sh, or a
// self-hosted instance) - a user provides their own topic URL in settings,
// and we POST to it. No account/auth with ntfy itself is required for the
// default public instance's topic-based model.
package notify

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// SendNtfy posts a message to an ntfy topic URL.
// https://docs.ntfy.sh/publish/
//
// The title is folded into the message body rather than sent as ntfy's
// `Title` header, since that header must be plain ASCII (RFC 2047 encoding
// otherwise) and titles here are often movie/show names with non-ASCII
// characters.
func SendNtfy(topicUrl string, message string) error {
	if topicUrl == "" {
		return nil
	}
	req, err := http.NewRequest(http.MethodPost, topicUrl, strings.NewReader(message))
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		slog.Error("notify.SendNtfy: request failed", "error", err)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		slog.Error("notify.SendNtfy: non-2xx response", "status", resp.StatusCode)
		return fmt.Errorf("ntfy responded with status %d", resp.StatusCode)
	}
	return nil
}

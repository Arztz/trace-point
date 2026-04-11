package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/trace-point/trace-point/internal/storage"
	"github.com/trace-point/trace-point/internal/utils/logger"
)

// Discord red color for embeds
const embedColor = 16711680

// Client handles Discord webhook notifications
type Client struct {
	httpClient  *http.Client
	webhookURL  string
	mentionUser string
	mentionRole string
	maxRetries  int
}

// WebhookMessage represents a Discord webhook payload
type WebhookMessage struct {
	Content   string  `json:"content,omitempty"`
	Embeds    []Embed `json:"embeds,omitempty"`
	Username  string  `json:"username,omitempty"`
	AvatarURL string  `json:"avatar_url,omitempty"`
}

// Embed represents a Discord embed
type Embed struct {
	Title       string  `json:"title"`
	Description string  `json:"description,omitempty"`
	Color       int     `json:"color"`
	Fields      []Field `json:"fields,omitempty"`
	Timestamp   string  `json:"timestamp"`
	Footer      *Footer `json:"footer,omitempty"`
}

// Field represents an embed field
type Field struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

// Footer represents an embed footer
type Footer struct {
	Text string `json:"text"`
}

// NewClient creates a new Discord client
func NewClient(webhookURL, mentionUser, mentionRole string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		webhookURL:  webhookURL,
		mentionUser: mentionUser,
		mentionRole: mentionRole,
		maxRetries:  3,
	}
}

// SendSpikeAlert sends a spike event alert to Discord
func (c *Client) SendSpikeAlert(ctx context.Context, event *storage.SpikeEvent) error {
	if c.webhookURL == "" {
		logger.Warn("Discord webhook URL not configured, skipping alert")
		return nil
	}

	msg := c.formatSpikeMessage(event)
	return c.sendWithRetry(ctx, msg)
}

// sendWithRetry sends a message with exponential backoff retry
func (c *Client) sendWithRetry(ctx context.Context, msg *WebhookMessage) error {
	var lastErr error

	for attempt := 0; attempt < c.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			time.Sleep(backoff)
		}

		err := c.send(ctx, msg)
		if err == nil {
			return nil
		}

		lastErr = err
		logger.Warn("Discord webhook attempt %d failed: %v", attempt+1, err)
	}

	return fmt.Errorf("failed to send Discord message after %d attempts: %w", c.maxRetries, lastErr)
}

// send sends a message to the Discord webhook
func (c *Client) send(ctx context.Context, msg *WebhookMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Discord webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// formatSpikeMessage formats a spike event as a Discord embed message
func (c *Client) formatSpikeMessage(event *storage.SpikeEvent) *WebhookMessage {
	var content string
	if c.mentionUser != "" {
		content = fmt.Sprintf("<@%s>", c.mentionUser)
	} else if c.mentionRole != "" {
		content = fmt.Sprintf("<@&%s>", c.mentionRole)
	}

	// Format CPU and RAM impact
	cpuImpact := formatResourceImpact(event.CPUUsagePercent, event.CPULimitPercent)
	ramImpact := formatResourceImpact(event.RAMUsagePercent, event.RAMLimitPercent)

	// Get culprit function
	culpritFunc := "Unknown"
	if event.CulpritFunction != nil && *event.CulpritFunction != "" {
		culpritFunc = *event.CulpritFunction
	}

	// Get trace ID
	traceID := "N/A"
	if event.TraceID != nil && *event.TraceID != "" {
		traceID = *event.TraceID
	}

	// Get route name
	routeName := "Unknown"
	if event.RouteName != nil && *event.RouteName != "" {
		routeName = *event.RouteName
	}

	fields := []Field{
		{Name: "Route", Value: routeName, Inline: true},
		{Name: "CPU Impact", Value: cpuImpact, Inline: true},
		{Name: "RAM Impact", Value: ramImpact, Inline: true},
		{Name: "Culprit Function", Value: culpritFunc, Inline: true},
		{Name: "Timestamp", Value: event.Timestamp.Format(time.RFC3339), Inline: true},
		{Name: "Trace ID", Value: traceID, Inline: true},
		{Name: "Pod", Value: fmt.Sprintf("%s (%s)", event.PodName, event.Namespace), Inline: false},
	}

	embed := Embed{
		Title:     fmt.Sprintf("[CRITICAL] Resource Spike Detected - %s", event.PodName),
		Color:     embedColor,
		Fields:    fields,
		Timestamp: time.Now().Format(time.RFC3339),
		Footer: &Footer{
			Text: "Trace-Point Alert System",
		},
	}

	return &WebhookMessage{
		Content:  content,
		Embeds:   []Embed{embed},
		Username: "Trace-Point Alerts",
	}
}

// formatResourceImpact formats resource usage as a readable string
func formatResourceImpact(usagePercent, limitPercent float64) string {
	if limitPercent > 0 {
		return fmt.Sprintf("%.1f%% / %.1f%% limit", usagePercent, limitPercent)
	}
	return fmt.Sprintf("%.1f%%", usagePercent)
}

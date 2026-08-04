package notify

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/iniwex5/vohive/internal/config"
)

const wxPusherSendEndpoint = "https://wxpusher.zjiecode.com/api/send/message"

type WXPusherChannel struct {
	cfg      config.WXPusherConfig
	client   *http.Client
	endpoint string
}

type wxPusherMessage struct {
	AppToken    string   `json:"appToken"`
	Content     string   `json:"content"`
	Summary     string   `json:"summary"`
	ContentType int      `json:"contentType"`
	UIDs        []string `json:"uids,omitempty"`
	TopicIDs    []int64  `json:"topicIds,omitempty"`
}

type wxPusherResponse struct {
	Code    int    `json:"code"`
	Message string `json:"msg"`
}

func NewWXPusherChannel(cfg config.WXPusherConfig) (*WXPusherChannel, error) {
	cfg.AppToken = strings.TrimSpace(cfg.AppToken)
	cfg.UIDs = compactStrings(cfg.UIDs)
	if cfg.AppToken == "" {
		return nil, errors.New("wxpusher app token is required")
	}
	if len(cfg.UIDs) == 0 && len(cfg.TopicIDs) == 0 {
		return nil, errors.New("wxpusher requires at least one UID or topic ID")
	}
	for _, topicID := range cfg.TopicIDs {
		if topicID <= 0 {
			return nil, fmt.Errorf("wxpusher topic ID must be positive: %d", topicID)
		}
	}

	return &WXPusherChannel{
		cfg:      cfg,
		client:   &http.Client{Timeout: 10 * time.Second},
		endpoint: wxPusherSendEndpoint,
	}, nil
}

func (c *WXPusherChannel) Name() string { return "wxpusher" }

func (c *WXPusherChannel) Send(text string) error {
	return c.SendWithContext(NotificationContext{Event: "notification", Text: text})
}

func (c *WXPusherChannel) SendWithContext(ctx NotificationContext) error {
	content := strings.TrimSpace(ctx.Text)
	if content == "" {
		return errors.New("wxpusher content is required")
	}

	body, err := json.Marshal(wxPusherMessage{
		AppToken:    c.cfg.AppToken,
		Content:     content,
		Summary:     wxPusherSummary(ctx),
		ContentType: 3,
		UIDs:        c.cfg.UIDs,
		TopicIDs:    c.cfg.TopicIDs,
	})
	if err != nil {
		return fmt.Errorf("marshal wxpusher message: %w", err)
	}

	endpoint := c.endpoint
	if endpoint == "" {
		endpoint = wxPusherSendEndpoint
	}
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create wxpusher request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := c.client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send wxpusher message: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return fmt.Errorf("read wxpusher response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("wxpusher returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	var result wxPusherResponse
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return fmt.Errorf("decode wxpusher response: %w", err)
	}
	if result.Code != 1000 {
		return fmt.Errorf("wxpusher returned business code %d: %s", result.Code, strings.TrimSpace(result.Message))
	}
	return nil
}

func (c *WXPusherChannel) RegisterCommand(_ string, _ CommandHandler) {}

func (c *WXPusherChannel) Start() error { return nil }

func (c *WXPusherChannel) Close() error { return nil }

func wxPusherSummary(ctx NotificationContext) string {
	summary := fmt.Sprintf("[Vohive] %s", strings.TrimSpace(ctx.Event))
	if label := ctx.DeviceLabel(); label != "未知设备" {
		summary += " - " + label
	}
	return truncateRunes(summary, 100)
}

func compactStrings(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			result = append(result, value)
		}
	}
	return result
}

func truncateRunes(value string, limit int) string {
	if limit <= 0 || utf8.RuneCountInString(value) <= limit {
		return value
	}
	return string([]rune(value)[:limit])
}

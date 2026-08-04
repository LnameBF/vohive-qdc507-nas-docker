package notify

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/iniwex5/vohive/internal/config"
)

const serverChanAPIBaseURL = "https://sctapi.ftqq.com"

type ServerChanChannel struct {
	cfg      config.ServerChanConfig
	client   *http.Client
	endpoint string
}

type serverChanMessage struct {
	Title   string `json:"title"`
	Desp    string `json:"desp,omitempty"`
	NoIP    *int   `json:"noip,omitempty"`
	Channel string `json:"channel,omitempty"`
	OpenID  string `json:"openid,omitempty"`
	Encoded *int   `json:"encoded,omitempty"`
}

type serverChanResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func NewServerChanChannel(cfg config.ServerChanConfig) (*ServerChanChannel, error) {
	cfg.SendKey = strings.TrimSpace(cfg.SendKey)
	cfg.Channel = strings.TrimSpace(cfg.Channel)
	cfg.OpenID = strings.TrimSpace(cfg.OpenID)
	cfg.EncryptionPassword = strings.TrimSpace(cfg.EncryptionPassword)
	cfg.EncryptionUID = strings.TrimSpace(cfg.EncryptionUID)
	if cfg.SendKey == "" {
		return nil, errors.New("serverchan send key is required")
	}
	if (cfg.EncryptionPassword == "") != (cfg.EncryptionUID == "") {
		return nil, errors.New("serverchan encryption password and UID must be provided together")
	}
	return &ServerChanChannel{
		cfg:    cfg,
		client: &http.Client{Timeout: 10 * time.Second},
	}, nil
}

func (c *ServerChanChannel) Name() string { return "serverchan" }

func (c *ServerChanChannel) Send(text string) error {
	return c.SendWithContext(NotificationContext{Event: "notification", Text: text})
}

func (c *ServerChanChannel) SendWithContext(ctx NotificationContext) error {
	desp := ctx.Text
	message := serverChanMessage{Title: serverChanTitle(ctx), Desp: desp}
	if c.cfg.HideIP {
		value := 1
		message.NoIP = &value
	}
	message.Channel = c.cfg.Channel
	message.OpenID = c.cfg.OpenID
	if c.cfg.EncryptionPassword != "" {
		encoded, err := encodeServerChanContent(desp, c.cfg.EncryptionPassword, c.cfg.EncryptionUID)
		if err != nil {
			return fmt.Errorf("encode serverchan content: %w", err)
		}
		value := 1
		message.Desp = encoded
		message.Encoded = &value
	}

	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshal serverchan message: %w", err)
	}
	endpoint := c.endpoint
	if endpoint == "" {
		endpoint = fmt.Sprintf("%s/%s.send", serverChanAPIBaseURL, url.PathEscape(c.cfg.SendKey))
	}
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create serverchan request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json;charset=utf-8")
	client := c.client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send serverchan message: %w", err)
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return fmt.Errorf("read serverchan response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("serverchan returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}
	var result serverChanResponse
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return fmt.Errorf("decode serverchan response: %w", err)
	}
	if result.Code != 0 {
		return fmt.Errorf("serverchan returned business code %d: %s", result.Code, strings.TrimSpace(result.Message))
	}
	return nil
}

func (c *ServerChanChannel) RegisterCommand(_ string, _ CommandHandler) {}

func (c *ServerChanChannel) Start() error { return nil }

func (c *ServerChanChannel) Close() error {
	if c != nil && c.client != nil {
		c.client.CloseIdleConnections()
	}
	return nil
}

func serverChanTitle(ctx NotificationContext) string {
	title := fmt.Sprintf("[Vohive] %s", strings.TrimSpace(ctx.Event))
	if label := ctx.DeviceLabel(); label != "未知设备" {
		title += " - " + label
	}
	return truncateRunes(title, 32)
}

// encodeServerChanContent matches Server 酱's documented PHP sc_encode routine.
func encodeServerChanContent(content, password, uid string) (string, error) {
	keyDigest := md5.Sum([]byte(password))
	ivDigest := md5.Sum([]byte("SCT" + uid))
	key := []byte(fmt.Sprintf("%x", keyDigest)[:16])
	iv := []byte(fmt.Sprintf("%x", ivDigest)[:16])
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	plain := []byte(base64.StdEncoding.EncodeToString([]byte(content)))
	padding := aes.BlockSize - len(plain)%aes.BlockSize
	plain = append(plain, bytes.Repeat([]byte{byte(padding)}, padding)...)
	ciphertext := make([]byte, len(plain))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(ciphertext, plain)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

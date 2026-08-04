package notify

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/iniwex5/vohive/internal/config"
)

func TestNewServerChanChannelValidatesConfiguration(t *testing.T) {
	for _, cfg := range []config.ServerChanConfig{
		{},
		{SendKey: "SCT_test", EncryptionPassword: "password"},
		{SendKey: "SCT_test", EncryptionUID: "227658"},
	} {
		if _, err := NewServerChanChannel(cfg); err == nil {
			t.Fatalf("NewServerChanChannel(%+v) error = nil, want validation error", cfg)
		}
	}
}

func TestServerChanChannelSendWithContext(t *testing.T) {
	var message serverChanMessage
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/SCT_test.send" {
			t.Errorf("path = %s, want /SCT_test.send", r.URL.Path)
		}
		if got := r.Header.Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
			t.Errorf("Content-Type = %q, want application/json", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_, _ = w.Write([]byte(`{"code":0,"message":"success"}`))
	}))
	defer server.Close()

	channel := &ServerChanChannel{
		cfg: config.ServerChanConfig{
			SendKey:            "SCT_test",
			Channel:            "9|66",
			HideIP:             true,
			OpenID:             "user-1",
			EncryptionPassword: "password",
			EncryptionUID:      "227658",
		},
		client:   server.Client(),
		endpoint: server.URL + "/SCT_test.send",
	}
	content := "短信正文 **Markdown**"
	if err := channel.SendWithContext(NotificationContext{Event: "sms_received", Text: content, DeviceID: "dev-1", DeviceName: "主设备"}); err != nil {
		t.Fatalf("SendWithContext() error = %v", err)
	}
	if message.Title == "" || len([]rune(message.Title)) > 32 {
		t.Errorf("title = %q, want a non-empty title no longer than 32 runes", message.Title)
	}
	if message.NoIP == nil || *message.NoIP != 1 {
		t.Errorf("noip = %#v, want 1", message.NoIP)
	}
	if message.Channel != "9|66" || message.OpenID != "user-1" {
		t.Errorf("channel/openid = %q/%q, want 9|66/user-1", message.Channel, message.OpenID)
	}
	if message.Encoded == nil || *message.Encoded != 1 {
		t.Errorf("encoded = %#v, want 1", message.Encoded)
	}
	if got := decodeServerChanContent(t, message.Desp, "password", "227658"); got != content {
		t.Errorf("decoded desp = %q, want %q", got, content)
	}
}

func TestServerChanChannelRejectsBusinessError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":1001,"message":"invalid key"}`))
	}))
	defer server.Close()
	channel := &ServerChanChannel{cfg: config.ServerChanConfig{SendKey: "SCT_test"}, client: server.Client(), endpoint: server.URL}
	if err := channel.Send("test"); err == nil || !strings.Contains(err.Error(), "1001") {
		t.Fatalf("Send() error = %v, want business error", err)
	}
}

func decodeServerChanContent(t *testing.T, encoded, password, uid string) string {
	t.Helper()
	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("DecodeString() error = %v", err)
	}
	keyDigest := md5.Sum([]byte(password))
	ivDigest := md5.Sum([]byte("SCT" + uid))
	key := []byte(fmt.Sprintf("%x", keyDigest)[:16])
	iv := []byte(fmt.Sprintf("%x", ivDigest)[:16])
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatalf("NewCipher() error = %v", err)
	}
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(ciphertext, ciphertext)
	padding := int(ciphertext[len(ciphertext)-1])
	plain, err := base64.StdEncoding.DecodeString(string(ciphertext[:len(ciphertext)-padding]))
	if err != nil {
		t.Fatalf("DecodeString plaintext() error = %v", err)
	}
	return string(plain)
}

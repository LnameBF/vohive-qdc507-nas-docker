package notify

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/iniwex5/vohive/internal/config"
)

func TestNewWXPusherChannelRequiresAppTokenAndRecipient(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.WXPusherConfig
	}{
		{
			name: "missing app token",
			cfg:  config.WXPusherConfig{UIDs: []string{"UID_test"}},
		},
		{
			name: "missing recipients",
			cfg:  config.WXPusherConfig{AppToken: "AT_test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewWXPusherChannel(tt.cfg); err == nil {
				t.Fatal("NewWXPusherChannel() error = nil, want validation error")
			}
		})
	}
}

func TestWXPusherChannelSendWithContext(t *testing.T) {
	var payload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/send/message" {
			t.Errorf("path = %s, want /api/send/message", r.URL.Path)
		}
		if got := r.Header.Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
			t.Errorf("Content-Type = %q, want application/json", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		_, _ = w.Write([]byte(`{"code":1000,"msg":"success"}`))
	}))
	defer server.Close()

	channel := &WXPusherChannel{
		cfg: config.WXPusherConfig{
			AppToken: "AT_test",
			UIDs:     []string{"UID_one", "UID_two"},
			TopicIDs: []int64{123, 456},
		},
		client:   server.Client(),
		endpoint: server.URL + "/api/send/message",
	}

	err := channel.SendWithContext(NotificationContext{
		Event:      "sms_received",
		Text:       "测试短信正文",
		DeviceID:   "dev-1",
		DeviceName: "主设备",
	})
	if err != nil {
		t.Fatalf("SendWithContext() error = %v", err)
	}

	if got := payload["appToken"]; got != "AT_test" {
		t.Errorf("appToken = %#v, want AT_test", got)
	}
	if got := payload["content"]; got != "测试短信正文" {
		t.Errorf("content = %#v, want 测试短信正文", got)
	}
	if got := payload["contentType"]; got != float64(3) {
		t.Errorf("contentType = %#v, want 3", got)
	}
	if got := payload["summary"]; got != "[Vohive] sms_received - 主设备 (dev-1)" {
		t.Errorf("summary = %#v, want contextual summary", got)
	}
	if got := payload["uids"]; !equalJSONArrays(got, []string{"UID_one", "UID_two"}) {
		t.Errorf("uids = %#v, want configured UIDs", got)
	}
	if got := payload["topicIds"]; !equalJSONArrays(got, []int64{123, 456}) {
		t.Errorf("topicIds = %#v, want configured topic IDs", got)
	}
}

func TestWXPusherChannelRejectsBusinessError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":1001,"msg":"invalid token"}`))
	}))
	defer server.Close()

	channel := &WXPusherChannel{
		cfg:      config.WXPusherConfig{AppToken: "AT_test", UIDs: []string{"UID_test"}},
		client:   server.Client(),
		endpoint: server.URL,
	}

	err := channel.Send("测试")
	if err == nil || !strings.Contains(err.Error(), "1001") {
		t.Fatalf("Send() error = %v, want WxPusher business error", err)
	}
}

func equalJSONArrays[T comparable](got any, want []T) bool {
	values, ok := got.([]any)
	if !ok || len(values) != len(want) {
		return false
	}
	for i, wantValue := range want {
		if values[i] != wantValue && values[i] != float64Value(wantValue) {
			return false
		}
	}
	return true
}

func float64Value[T comparable](value T) any {
	switch value := any(value).(type) {
	case int64:
		return float64(value)
	default:
		return value
	}
}

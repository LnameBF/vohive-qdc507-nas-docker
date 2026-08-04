package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/iniwex5/vohive/internal/config"
	"github.com/iniwex5/vohive/internal/notify"
)

type testServerChanRequest struct {
	Enabled            bool   `json:"enabled"`
	SendKey            string `json:"send_key"`
	Channel            string `json:"channel"`
	HideIP             bool   `json:"hide_ip"`
	OpenID             string `json:"openid"`
	EncryptionPassword string `json:"encryption_password"`
	EncryptionUID      string `json:"encryption_uid"`
}

type testServerChanResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

func (s *Server) handleTestServerChanNotification(c *gin.Context) {
	var req testServerChanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "参数错误"})
		return
	}
	if !req.Enabled {
		c.JSON(http.StatusBadRequest, gin.H{"message": "请先启用 Server 酱后再测试"})
		return
	}
	ch, err := notify.NewServerChanChannel(config.ServerChanConfig{
		Enabled:            true,
		SendKey:            strings.TrimSpace(req.SendKey),
		Channel:            strings.TrimSpace(req.Channel),
		HideIP:             req.HideIP,
		OpenID:             strings.TrimSpace(req.OpenID),
		EncryptionPassword: strings.TrimSpace(req.EncryptionPassword),
		EncryptionUID:      strings.TrimSpace(req.EncryptionUID),
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Server 酱配置无效: " + err.Error()})
		return
	}
	defer ch.Close()
	if err := ch.SendWithContext(notify.NotificationContext{
		Event:      "serverchan_test",
		Text:       "这是一条 Server 酱测试通知",
		DeviceID:   "test_device_001",
		DeviceName: "测试设备",
		Timestamp:  time.Now(),
	}); err != nil {
		c.JSON(http.StatusOK, testServerChanResponse{OK: false, Message: "测试通知发送失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, testServerChanResponse{OK: true, Message: "测试通知已提交到 Server 酱推送队列"})
}

package handlers

import (
	"crypto/rand"

	"encoding/hex"
	"fmt"
	"log"
	"net/http"

	"strconv"
	"strings"
	"subtrackr/internal/middleware"
	"subtrackr/internal/models"
	"subtrackr/internal/service"
	"time"

	"github.com/gin-gonic/gin"
)

func splitLines(s string) []string         { return strings.Split(s, "\n") }
func trimSpace(s string) string            { return strings.TrimSpace(s) }
func splitN(s, sep string, n int) []string { return strings.SplitN(s, sep, n) }

type SettingsHandler struct {
	service *service.SettingsService
}

func NewSettingsHandler(service *service.SettingsService) *SettingsHandler {
	return &SettingsHandler{service: service}
}

func (h *SettingsHandler) scopedService(c *gin.Context) *service.SettingsService {
	return h.service.ForUser(middleware.CurrentUserID(c))
}

func (h *SettingsHandler) UpdateNotificationSetting(c *gin.Context) {
	svc := h.scopedService(c)
	setting := c.Param("setting")

	switch setting {
	case "renewal":
		current, _ := svc.GetBoolSetting("renewal_reminders", false)
		err := svc.SetBoolSetting("renewal_reminders", !current)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"enabled": !current})
	case "highcost":
		current, _ := svc.GetBoolSetting("high_cost_alerts", true)
		err := svc.SetBoolSetting("high_cost_alerts", !current)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"enabled": !current})
	case "days":
		daysStr := c.PostForm("reminder_days")
		if days, err := strconv.Atoi(daysStr); err == nil && days > 0 && days <= 30 {
			if err := svc.SetIntSetting("reminder_days", days); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"days": days})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid days value"})
		}
	case "threshold":
		thresholdStr := c.PostForm("high_cost_threshold")
		if threshold, err := strconv.ParseFloat(thresholdStr, 64); err == nil && threshold >= 0 && threshold <= 10000 {
			if err := svc.SetFloatSetting("high_cost_threshold", threshold); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"threshold": threshold})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid threshold value (must be between 0 and 10000)"})
		}
	case "cancellation":
		current, _ := svc.GetBoolSetting("cancellation_reminders", false)
		err := svc.SetBoolSetting("cancellation_reminders", !current)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"enabled": !current})
	case "cancellation_days":
		daysStr := c.PostForm("cancellation_reminder_days")
		if days, err := strconv.Atoi(daysStr); err == nil && days > 0 && days <= 30 {
			if err := svc.SetIntSetting("cancellation_reminder_days", days); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"days": days})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid days value"})
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unknown setting"})
	}
}

func (h *SettingsHandler) GetNotificationSettings(c *gin.Context) {
	svc := h.scopedService(c)
	settings := models.NotificationSettings{
		RenewalReminders:         svc.GetBoolSettingWithDefault("renewal_reminders", false),
		HighCostAlerts:           svc.GetBoolSettingWithDefault("high_cost_alerts", true),
		HighCostThreshold:        svc.GetFloatSettingWithDefault("high_cost_threshold", 50.0),
		ReminderDays:             svc.GetIntSettingWithDefault("reminder_days", 7),
		CancellationReminders:    svc.GetBoolSettingWithDefault("cancellation_reminders", false),
		CancellationReminderDays: svc.GetIntSettingWithDefault("cancellation_reminder_days", 7),
	}
	c.JSON(http.StatusOK, settings)
}

func (h *SettingsHandler) SaveUIPersonalizationSettings(c *gin.Context) {
	svc := h.scopedService(c)
	config := &models.UIPersonalizationConfig{
		CustomBackgroundURL: strings.TrimSpace(c.PostForm("custom_background_url")),
		EnableChibiStickers: c.PostForm("enable_chibi_stickers") == "on",
		ReduceMotion:        c.PostForm("reduce_motion") == "on",
		StaticStickersOnly:  c.PostForm("static_stickers_only") == "on",
	}

	if config.CustomBackgroundURL != "" && !strings.HasPrefix(config.CustomBackgroundURL, "http://") && !strings.HasPrefix(config.CustomBackgroundURL, "https://") {
		c.HTML(http.StatusBadRequest, "settings-message.html", gin.H{"Error": "Background image URL must use http:// or https://", "Type": "error"})
		return
	}

	if err := svc.SaveUIPersonalizationConfig(config); err != nil {
		c.HTML(http.StatusInternalServerError, "settings-message.html", gin.H{"Error": err.Error(), "Type": "error"})
		return
	}

	c.HTML(http.StatusOK, "settings-message.html", gin.H{"Message": "UI personalization saved successfully", "Type": "success"})
}

func (h *SettingsHandler) GetUIPersonalizationSettings(c *gin.Context) {
	svc := h.scopedService(c)
	config, err := svc.GetUIPersonalizationConfig()
	if err != nil && config == nil {
		c.JSON(http.StatusOK, service.DefaultUIPersonalizationConfig)
		return
	}
	c.JSON(http.StatusOK, config)
}

func (h *SettingsHandler) ListAPIKeys(c *gin.Context) {
	svc := h.scopedService(c)
	keys, err := svc.GetAllAPIKeys()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "api-keys-list.html", gin.H{"Error": err.Error()})
		return
	}
	for i := range keys {
		if !keys[i].IsNew {
			keys[i].Key = ""
		}
	}
	c.HTML(http.StatusOK, "api-keys-list.html", gin.H{"Keys": keys, "GoDateFormat": svc.GetGoDateFormat()})
}

func (h *SettingsHandler) CreateAPIKey(c *gin.Context) {
	svc := h.scopedService(c)
	name := c.PostForm("name")
	if name == "" {
		c.HTML(http.StatusBadRequest, "api-keys-list.html", gin.H{"Error": "API key name is required"})
		return
	}

	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		c.HTML(http.StatusInternalServerError, "api-keys-list.html", gin.H{"Error": "Failed to generate API key"})
		return
	}

	apiKey := "sk_" + hex.EncodeToString(keyBytes)
	newKey, err := svc.CreateAPIKey(name, apiKey)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "api-keys-list.html", gin.H{"Error": err.Error()})
		return
	}

	keys, err := svc.GetAllAPIKeys()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "api-keys-list.html", gin.H{"Error": err.Error()})
		return
	}

	for i := range keys {
		if keys[i].ID == newKey.ID {
			keys[i].IsNew = true
			keys[i].Key = apiKey
		} else {
			keys[i].Key = ""
		}
	}

	c.HTML(http.StatusOK, "api-keys-list.html", gin.H{"Keys": keys, "GoDateFormat": svc.GetGoDateFormat()})
}

func (h *SettingsHandler) DeleteAPIKey(c *gin.Context) {
	svc := h.scopedService(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.HTML(http.StatusBadRequest, "api-keys-list.html", gin.H{"Error": "Invalid API key ID"})
		return
	}
	if err := svc.DeleteAPIKey(uint(id)); err != nil {
		c.HTML(http.StatusInternalServerError, "api-keys-list.html", gin.H{"Error": err.Error()})
		return
	}
	h.ListAPIKeys(c)
}

func (h *SettingsHandler) UpdateCurrency(c *gin.Context) {
	svc := h.scopedService(c)
	currency := c.PostForm("currency")
	if err := svc.SetCurrency(currency); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"currency": currency, "symbol": svc.GetCurrencySymbol()})
}

func (h *SettingsHandler) UpdateDateFormat(c *gin.Context) {
	svc := h.scopedService(c)
	format := c.PostForm("date_format")
	if err := svc.SetDateFormat(format); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"date_format": format})
}

func (h *SettingsHandler) ToggleDarkMode(c *gin.Context) {
	svc := h.scopedService(c)
	enabled := c.PostForm("enabled") == "true"
	if err := svc.SetDarkMode(enabled); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"dark_mode": enabled})
}

func (h *SettingsHandler) GetTheme(c *gin.Context) {
	svc := h.scopedService(c)
	theme, err := svc.GetTheme()
	if err != nil {
		theme = "default"
	}
	c.JSON(http.StatusOK, gin.H{"theme": theme})
}

func (h *SettingsHandler) SaveWebhookSettings(c *gin.Context) {
	svc := h.scopedService(c)
	var config models.WebhookConfig
	config.URL = c.PostForm("webhook_url")
	if config.URL == "" {
		c.HTML(http.StatusBadRequest, "settings-message.html", gin.H{"Error": "Webhook URL is required", "Type": "error"})
		return
	}
	if !strings.HasPrefix(config.URL, "http://") && !strings.HasPrefix(config.URL, "https://") {
		c.HTML(http.StatusBadRequest, "settings-message.html", gin.H{"Error": "Webhook URL must use http:// or https:// scheme", "Type": "error"})
		return
	}
	headers := make(map[string]string)
	for _, line := range splitLines(c.PostForm("webhook_headers")) {
		line = trimSpace(line)
		if line == "" {
			continue
		}
		parts := splitN(line, ":", 2)
		if len(parts) == 2 {
			headers[trimSpace(parts[0])] = trimSpace(parts[1])
		}
	}
	config.Headers = headers
	if err := svc.SaveWebhookConfig(&config); err != nil {
		c.HTML(http.StatusInternalServerError, "settings-message.html", gin.H{"Error": err.Error(), "Type": "error"})
		return
	}
	c.HTML(http.StatusOK, "settings-message.html", gin.H{"Message": "Webhook settings saved successfully", "Type": "success"})
}

func (h *SettingsHandler) TestWebhookConnection(c *gin.Context) {
	svc := h.scopedService(c)
	webhookURL := c.PostForm("webhook_url")
	if webhookURL == "" {
		c.HTML(http.StatusBadRequest, "settings-message.html", gin.H{"Error": "Webhook URL is required for testing", "Type": "error"})
		return
	}
	if !strings.HasPrefix(webhookURL, "http://") && !strings.HasPrefix(webhookURL, "https://") {
		c.HTML(http.StatusBadRequest, "settings-message.html", gin.H{"Error": "Webhook URL must use http:// or https:// scheme", "Type": "error"})
		return
	}

	headers := make(map[string]string)
	for _, line := range splitLines(c.PostForm("webhook_headers")) {
		line = trimSpace(line)
		if line == "" {
			continue
		}
		parts := splitN(line, ":", 2)
		if len(parts) == 2 {
			headers[trimSpace(parts[0])] = trimSpace(parts[1])
		}
	}

	testConfig := &models.WebhookConfig{URL: webhookURL, Headers: headers}
	originalConfig, _ := svc.GetWebhookConfig()
	defer func() {
		var restoreErr error
		if originalConfig != nil {
			restoreErr = svc.SaveWebhookConfig(originalConfig)
		} else {
			restoreErr = svc.SaveWebhookConfig(&models.WebhookConfig{})
		}
		if restoreErr != nil {
			log.Printf("Warning: failed to restore webhook config after test: %v", restoreErr)
		}
	}()

	if err := svc.SaveWebhookConfig(testConfig); err != nil {
		c.HTML(http.StatusBadRequest, "settings-message.html", gin.H{"Error": fmt.Sprintf("Failed to save test config: %v", err), "Type": "error"})
		return
	}

	webhookService := service.NewWebhookService(svc)
	payload := &service.WebhookPayload{
		Event:     "test",
		Title:     "yyTrackr 测试消息",
		Message:   "这是一条来自 yyTrackr 的测试通知。如果你收到了这条消息，说明当前 Webhook 配置工作正常。",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	if err := webhookService.SendWebhook(payload); err != nil {
		c.HTML(http.StatusBadRequest, "settings-message.html", gin.H{"Error": fmt.Sprintf("Webhook test failed: %v", err), "Type": "error"})
		return
	}
	c.HTML(http.StatusOK, "settings-message.html", gin.H{"Message": "Webhook 测试成功，请检查目标端点是否已收到通知。", "Type": "success"})
}

func (h *SettingsHandler) SaveTelegramSettings(c *gin.Context) {
	svc := h.scopedService(c)
	config := models.TelegramConfig{
		BotToken: c.PostForm("telegram_bot_token"),
		ChatID:   c.PostForm("telegram_chat_id"),
	}
	if config.BotToken == "" || config.ChatID == "" {
		c.HTML(http.StatusBadRequest, "settings-message.html", gin.H{"Error": "Bot token and chat ID are required", "Type": "error"})
		return
	}
	if err := svc.SaveTelegramConfig(&config); err != nil {
		c.HTML(http.StatusInternalServerError, "settings-message.html", gin.H{"Error": err.Error(), "Type": "error"})
		return
	}
	c.HTML(http.StatusOK, "settings-message.html", gin.H{"Message": "Telegram settings saved successfully", "Type": "success"})
}

func (h *SettingsHandler) TestTelegramConnection(c *gin.Context) {
	svc := h.scopedService(c)
	config := models.TelegramConfig{
		BotToken: c.PostForm("telegram_bot_token"),
		ChatID:   c.PostForm("telegram_chat_id"),
	}
	if config.BotToken == "" || config.ChatID == "" {
		c.HTML(http.StatusBadRequest, "settings-message.html", gin.H{"Error": "Bot token and chat ID are required for testing", "Type": "error"})
		return
	}

	telegramService := service.NewTelegramService(svc)
	originalConfig, _ := svc.GetTelegramConfig()
	defer func() {
		var restoreErr error
		if originalConfig != nil {
			restoreErr = svc.SaveTelegramConfig(originalConfig)
		} else {
			restoreErr = svc.SaveTelegramConfig(&models.TelegramConfig{})
		}
		if restoreErr != nil {
			log.Printf("Warning: failed to restore Telegram config after test: %v", restoreErr)
		}
	}()

	if err := svc.SaveTelegramConfig(&config); err != nil {
		c.HTML(http.StatusBadRequest, "settings-message.html", gin.H{"Error": fmt.Sprintf("Failed to save test config: %v", err), "Type": "error"})
		return
	}

	if err := telegramService.SendNotification("yyTrackr 测试消息", "这是一条来自 yyTrackr 的测试通知。如果你收到了这条消息，说明当前 Telegram 配置工作正常。"); err != nil {
		c.HTML(http.StatusBadRequest, "settings-message.html", gin.H{"Error": fmt.Sprintf("Failed to send Telegram notification: %v", err), "Type": "error"})
		return
	}
	c.HTML(http.StatusOK, "settings-message.html", gin.H{"Message": "Telegram 测试成功，请检查目标聊天是否已收到消息。", "Type": "success"})
}

func (h *SettingsHandler) GetTelegramConfig(c *gin.Context) {
	svc := h.scopedService(c)
	config, err := svc.GetTelegramConfig()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"configured": false})
		return
	}

	hasBotToken := config.BotToken != ""
	maskedToken := ""
	if hasBotToken {
		if len(config.BotToken) > 8 {
			maskedToken = config.BotToken[:4] + "..." + config.BotToken[len(config.BotToken)-4:]
		} else {
			maskedToken = "configured"
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"configured":        hasBotToken && config.ChatID != "",
		"has_bot_token":     hasBotToken,
		"bot_token_preview": maskedToken,
		"chat_id":           config.ChatID,
	})
}

func (h *SettingsHandler) SetTheme(c *gin.Context) {
	svc := h.scopedService(c)
	var req struct {
		Theme string `json:"theme" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	validThemes := map[string]bool{
		"default":      true,
		"dark":         true,
		"dark-classic": true,
		"christmas":    true,
		"midnight":     true,
		"ocean":        true,
		"gal-violet":   true,
	}
	if !validThemes[req.Theme] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid theme name"})
		return
	}

	if err := svc.SetTheme(req.Theme); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save theme"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "theme": req.Theme})
}

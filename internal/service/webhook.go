package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"subtrackr/internal/models"
	"time"
)

type WebhookService struct {
	settingsService *SettingsService
}

func NewWebhookService(settingsService *SettingsService) *WebhookService {
	return &WebhookService{
		settingsService: settingsService,
	}
}

type WebhookPayload struct {
	Event        string               `json:"event"`
	Title        string               `json:"title"`
	Message      string               `json:"message"`
	Subscription *WebhookSubscription `json:"subscription"`
	Timestamp    string               `json:"timestamp"`
}

type WebhookSubscription struct {
	ID               uint    `json:"id"`
	Name             string  `json:"name"`
	Cost             float64 `json:"cost"`
	Currency         string  `json:"currency"`
	CurrencySymbol   string  `json:"currency_symbol"`
	Schedule         string  `json:"schedule"`
	MonthlyCost      float64 `json:"monthly_cost"`
	Category         string  `json:"category,omitempty"`
	URL              string  `json:"url,omitempty"`
	RenewalDate      string  `json:"renewal_date,omitempty"`
	CancellationDate string  `json:"cancellation_date,omitempty"`
}

type discordWebhookEmbed struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Color       int    `json:"color,omitempty"`
	Timestamp   string `json:"timestamp,omitempty"`
}

type discordWebhookPayload struct {
	Username string                `json:"username,omitempty"`
	Content  string                `json:"content,omitempty"`
	Embeds   []discordWebhookEmbed `json:"embeds,omitempty"`
}

func subscriptionToWebhook(sub *models.Subscription, settings *SettingsService) *WebhookSubscription {
	currencySymbol := currencySymbolForSubscription(sub, settings)
	ws := &WebhookSubscription{
		ID:             sub.ID,
		Name:           sub.Name,
		Cost:           sub.Cost,
		Currency:       sub.OriginalCurrency,
		CurrencySymbol: currencySymbol,
		Schedule:       sub.Schedule,
		MonthlyCost:    sub.MonthlyCost(),
	}
	if sub.Category.Name != "" {
		ws.Category = sub.Category.Name
	}
	if sub.URL != "" {
		ws.URL = sub.URL
	}
	dateFormat := settings.GetGoDateFormat()
	if sub.RenewalDate != nil {
		ws.RenewalDate = sub.RenewalDate.Format(dateFormat)
	}
	if sub.CancellationDate != nil {
		ws.CancellationDate = sub.CancellationDate.Format(dateFormat)
	}
	return ws
}

func (w *WebhookService) SendWebhook(payload *WebhookPayload) error {
	config, err := w.settingsService.GetWebhookConfig()
	if err != nil || config.URL == "" {
		return nil
	}

	jsonData, err := marshalWebhookPayload(config.URL, payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	req, err := http.NewRequest("POST", config.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "yyTrackr-Webhook/1.0")

	for key, value := range config.Headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		if trimmed := strings.TrimSpace(string(body)); trimmed != "" {
			return fmt.Errorf("webhook returned status %d: %s", resp.StatusCode, trimmed)
		}
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

func marshalWebhookPayload(webhookURL string, payload *WebhookPayload) ([]byte, error) {
	if isDiscordWebhookURL(webhookURL) {
		return json.Marshal(buildDiscordWebhookPayload(payload))
	}
	return json.Marshal(payload)
}

func isDiscordWebhookURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Host)
	return strings.Contains(host, "discord.com") || strings.Contains(host, "discordapp.com")
}

func buildDiscordWebhookPayload(payload *WebhookPayload) *discordWebhookPayload {
	content := payload.Title
	if payload.Message != "" {
		content = payload.Title + "\n" + payload.Message
	}

	if payload.Subscription != nil {
		content += fmt.Sprintf(
			"\n\n订阅：%s\n费用：%s%.2f / %s",
			payload.Subscription.Name,
			payload.Subscription.CurrencySymbol,
			payload.Subscription.Cost,
			payload.Subscription.Schedule,
		)
		if payload.Subscription.RenewalDate != "" {
			content += "\n续费日期：" + payload.Subscription.RenewalDate
		}
		if payload.Subscription.CancellationDate != "" {
			content += "\n到期日期：" + payload.Subscription.CancellationDate
		}
	}

	return &discordWebhookPayload{
		Username: "yyTrackr",
		Content:  content,
		Embeds: []discordWebhookEmbed{
			{
				Title:       payload.Title,
				Description: payload.Message,
				Color:       0xF472B6,
				Timestamp:   payload.Timestamp,
			},
		},
	}
}

func (w *WebhookService) SendHighCostAlert(subscription *models.Subscription) error {
	enabled, err := w.settingsService.GetBoolSetting("high_cost_alerts", true)
	if err != nil || !enabled {
		return nil
	}

	payload := &WebhookPayload{
		Event:        "high_cost_alert",
		Title:        "高价订阅提醒",
		Message:      "检测到高价订阅，请确认是否继续保留。",
		Subscription: subscriptionToWebhook(subscription, w.settingsService),
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
	}

	return w.SendWebhook(payload)
}

func (w *WebhookService) SendRenewalReminder(subscription *models.Subscription, daysUntilRenewal int) error {
	enabled, err := w.settingsService.GetBoolSetting("renewal_reminders", false)
	if err != nil || !enabled {
		return nil
	}

	payload := &WebhookPayload{
		Event:        "renewal_reminder",
		Title:        fmt.Sprintf("续费提醒｜%s", subscription.Name),
		Message:      fmt.Sprintf("%s续费。", formatReminderDetail(daysUntilRenewal, "续费")),
		Subscription: subscriptionToWebhook(subscription, w.settingsService),
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
	}

	return w.SendWebhook(payload)
}

func (w *WebhookService) SendCancellationReminder(subscription *models.Subscription, daysUntilCancellation int) error {
	enabled, err := w.settingsService.GetBoolSetting("cancellation_reminders", false)
	if err != nil || !enabled {
		return nil
	}

	payload := &WebhookPayload{
		Event:        "cancellation_reminder",
		Title:        fmt.Sprintf("到期提醒｜%s", subscription.Name),
		Message:      fmt.Sprintf("%s到期。", formatReminderDetail(daysUntilCancellation, "到期")),
		Subscription: subscriptionToWebhook(subscription, w.settingsService),
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
	}

	return w.SendWebhook(payload)
}

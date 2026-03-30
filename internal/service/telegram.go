package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"subtrackr/internal/models"
	"time"
)

type TelegramService struct {
	settingsService *SettingsService
	httpClient      *http.Client
}

func NewTelegramService(settingsService *SettingsService) *TelegramService {
	return &TelegramService{
		settingsService: settingsService,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

type telegramSendMessageRequest struct {
	ChatID                string `json:"chat_id"`
	Text                  string `json:"text"`
	ParseMode             string `json:"parse_mode,omitempty"`
	DisableWebPagePreview bool   `json:"disable_web_page_preview"`
}

type telegramAPIResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description"`
}

func (t *TelegramService) SendNotification(title, message string) error {
	config, err := t.settingsService.GetTelegramConfig()
	if err != nil {
		return fmt.Errorf("failed to get Telegram config: %w", err)
	}

	if config.BotToken == "" || config.ChatID == "" {
		return fmt.Errorf("Telegram not configured: bot token and chat ID required")
	}

	body, err := json.Marshal(telegramSendMessageRequest{
		ChatID:                config.ChatID,
		Text:                  fmt.Sprintf("%s\n\n%s", title, message),
		DisableWebPagePreview: true,
	})
	if err != nil {
		return fmt.Errorf("failed to encode Telegram request: %w", err)
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", config.BotToken)
	req, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create Telegram request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Telegram notification: %w", err)
	}
	defer resp.Body.Close()

	var telegramResp telegramAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&telegramResp); err != nil {
		return fmt.Errorf("failed to decode Telegram response: %w", err)
	}

	if !telegramResp.OK {
		if telegramResp.Description == "" {
			telegramResp.Description = "unknown Telegram API error"
		}
		return fmt.Errorf("Telegram API error: %s", telegramResp.Description)
	}

	return nil
}

func (t *TelegramService) SendHighCostAlert(subscription *models.Subscription) error {
	enabled, err := t.settingsService.GetBoolSetting("high_cost_alerts", true)
	if err != nil || !enabled {
		return nil
	}

	message := joinNotificationLines(
		"检测到高价订阅，请确认是否需要保留。",
		joinNotificationLines(buildNotificationLines(subscription, t.settingsService)...),
	)

	return t.SendNotification("高价订阅提醒", message)
}

func (t *TelegramService) SendRenewalReminder(subscription *models.Subscription, daysUntilRenewal int) error {
	enabled, err := t.settingsService.GetBoolSetting("renewal_reminders", false)
	if err != nil || !enabled {
		return nil
	}

	message := joinNotificationLines(
		fmt.Sprintf("%s续费。", formatReminderDetail(daysUntilRenewal, "续费")),
		joinNotificationLines(buildNotificationLines(subscription, t.settingsService)...),
		func() string {
			if subscription.RenewalDate == nil {
				return ""
			}
			return "续费日期：" + subscription.RenewalDate.Format(t.settingsService.GetGoDateFormatLong())
		}(),
	)

	return t.SendNotification(fmt.Sprintf("续费提醒｜%s", subscription.Name), message)
}

func (t *TelegramService) SendCancellationReminder(subscription *models.Subscription, daysUntilCancellation int) error {
	enabled, err := t.settingsService.GetBoolSetting("cancellation_reminders", false)
	if err != nil || !enabled {
		return nil
	}

	message := joinNotificationLines(
		fmt.Sprintf("%s到期或取消。", formatReminderDetail(daysUntilCancellation, "到期")),
		joinNotificationLines(buildNotificationLines(subscription, t.settingsService)...),
		func() string {
			if subscription.CancellationDate == nil {
				return ""
			}
			return "到期日期：" + subscription.CancellationDate.Format(t.settingsService.GetGoDateFormatLong())
		}(),
	)

	return t.SendNotification(fmt.Sprintf("到期提醒｜%s", subscription.Name), message)
}

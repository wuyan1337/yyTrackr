package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"subtrackr/internal/models"
	"time"
)

type PushoverService struct {
	settingsService *SettingsService
}

func NewPushoverService(settingsService *SettingsService) *PushoverService {
	return &PushoverService{
		settingsService: settingsService,
	}
}

type PushoverResponse struct {
	Status  int      `json:"status"`
	Request string   `json:"request"`
	Errors  []string `json:"errors,omitempty"`
}

func (p *PushoverService) SendNotification(title, message string, priority int) error {
	config, err := p.settingsService.GetPushoverConfig()
	if err != nil {
		return fmt.Errorf("failed to get Pushover config: %w", err)
	}

	if config.UserKey == "" || config.AppToken == "" {
		return fmt.Errorf("Pushover not configured: user key and app token required")
	}

	apiURL := "https://api.pushover.net/1/messages.json"

	formData := url.Values{}
	formData.Set("token", config.AppToken)
	formData.Set("user", config.UserKey)
	formData.Set("title", title)
	formData.Set("message", message)
	formData.Set("priority", strconv.Itoa(priority))

	req, err := http.NewRequest("POST", apiURL, bytes.NewBufferString(formData.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Pushover notification: %w", err)
	}
	defer resp.Body.Close()

	var pushoverResp PushoverResponse
	if err := json.NewDecoder(resp.Body).Decode(&pushoverResp); err != nil {
		return fmt.Errorf("failed to decode Pushover response: %w", err)
	}

	if pushoverResp.Status != 1 {
		errorMsg := "Pushover API error"
		if len(pushoverResp.Errors) > 0 {
			errorMsg = pushoverResp.Errors[0]
		}
		return fmt.Errorf("%s", errorMsg)
	}

	return nil
}

func (p *PushoverService) SendHighCostAlert(subscription *models.Subscription) error {
	enabled, err := p.settingsService.GetBoolSetting("high_cost_alerts", true)
	if err != nil || !enabled {
		return nil
	}

	message := joinNotificationLines(
		"检测到高价订阅，请尽快确认。",
		joinNotificationLines(buildNotificationLines(subscription, p.settingsService)...),
	)

	return p.SendNotification("高价订阅提醒", message, 1)
}

func (p *PushoverService) SendRenewalReminder(subscription *models.Subscription, daysUntilRenewal int) error {
	enabled, err := p.settingsService.GetBoolSetting("renewal_reminders", false)
	if err != nil || !enabled {
		return nil
	}

	message := joinNotificationLines(
		fmt.Sprintf("%s续费。", formatReminderDetail(daysUntilRenewal, "续费")),
		joinNotificationLines(buildNotificationLines(subscription, p.settingsService)...),
		func() string {
			if subscription.RenewalDate == nil {
				return ""
			}
			return "续费日期：" + subscription.RenewalDate.Format(p.settingsService.GetGoDateFormatLong())
		}(),
	)

	return p.SendNotification(fmt.Sprintf("续费提醒｜%s", subscription.Name), message, 0)
}

func (p *PushoverService) SendCancellationReminder(subscription *models.Subscription, daysUntilCancellation int) error {
	enabled, err := p.settingsService.GetBoolSetting("cancellation_reminders", false)
	if err != nil || !enabled {
		return nil
	}

	message := joinNotificationLines(
		fmt.Sprintf("%s到期。", formatReminderDetail(daysUntilCancellation, "到期")),
		joinNotificationLines(buildNotificationLines(subscription, p.settingsService)...),
		func() string {
			if subscription.CancellationDate == nil {
				return ""
			}
			return "到期日期：" + subscription.CancellationDate.Format(p.settingsService.GetGoDateFormatLong())
		}(),
	)

	return p.SendNotification(fmt.Sprintf("到期提醒｜%s", subscription.Name), message, 0)
}

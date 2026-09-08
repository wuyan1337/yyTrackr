package service

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"subtrackr/internal/models"
	"subtrackr/internal/repository"
	"testing"
)

func TestNotificationDetailsRegression(t *testing.T) {
	db := setupRenewalReminderTestDB(t)
	settings := NewSettingsService(repository.NewSettingsRepository(db))
	sub := &models.Subscription{Name: "测试订阅", Cost: 12, Schedule: "Monthly", OriginalCurrency: "CNY", PaymentMethod: "支付宝", Notes: "第一行\n<script>alert(1)</script>", URL: "https://example.com/billing"}
	text := joinNotificationLines(buildNotificationLines(sub, settings)...)
	for _, want := range []string{"支付方式：支付宝", "备注：第一行", sub.URL} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q in %s", want, text)
		}
	}
	body, err := renderNotificationEmail("到期提醒", "明天到期", sub, settings, "到期日期", "2027-01-01")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"支付方式：", "支付宝", "备注：", "第一行", sub.URL, "&lt;script&gt;"} {
		if !strings.Contains(body, want) {
			t.Errorf("email missing %q", want)
		}
	}
	if strings.Contains(body, "<script>") {
		t.Fatal("unescaped notes")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload WebhookPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)
			return
		}
		for _, want := range []string{"支付方式：支付宝", "备注：第一行", sub.URL} {
			if !strings.Contains(payload.Message, want) {
				t.Errorf("webhook missing %q", want)
			}
		}
		discord := buildDiscordWebhookPayload(&payload)
		if !strings.Contains(discord.Content, "支付宝") || !strings.Contains(discord.Content, sub.URL) {
			t.Error("discord details missing")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	settings.SaveWebhookConfig(&models.WebhookConfig{URL: server.URL})
	settings.SetBoolSetting("renewal_reminders", true)
	settings.SetBoolSetting("cancellation_reminders", true)
	webhook := NewWebhookService(settings)
	if err := webhook.SendRenewalReminder(sub, 1); err != nil {
		t.Fatal(err)
	}
	if err := webhook.SendCancellationReminder(sub, 1); err != nil {
		t.Fatal(err)
	}
	empty := joinNotificationLines(buildNotificationLines(&models.Subscription{}, settings)...)
	for _, label := range []string{"备注：", "支付方式：", "链接："} {
		if strings.Contains(empty, label) {
			t.Errorf("empty field rendered: %s", label)
		}
	}
}

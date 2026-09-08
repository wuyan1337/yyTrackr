package service

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"
	"net/smtp"
	"subtrackr/internal/models"
)

func currencySymbolForSubscription(subscription *models.Subscription, settings *SettingsService) string {
	preferred := settings.GetCurrency()
	if subscription.OriginalCurrency != "" && subscription.OriginalCurrency != preferred {
		return CurrencySymbolForCode(subscription.OriginalCurrency)
	}
	return settings.GetCurrencySymbol()
}

type EmailService struct {
	settingsService *SettingsService
}

func NewEmailService(settingsService *SettingsService) *EmailService {
	return &EmailService{
		settingsService: settingsService,
	}
}

func (e *EmailService) SendEmail(subject, body string) error {
	config, err := e.settingsService.GetSMTPConfig()
	if err != nil {
		return fmt.Errorf("failed to get SMTP config: %w", err)
	}

	if config.To == "" {
		return fmt.Errorf("no recipient email configured")
	}

	isSSLPort := config.Port == 465 || config.Port == 8465 || config.Port == 443

	auth := smtp.PlainAuth("", config.Username, config.Password, config.Host)
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)

	buildMessage := func() string {
		fromName := config.FromName
		if fromName == "" {
			fromName = "yyTrackr"
		}

		message := fmt.Sprintf("From: %s <%s>\r\n", fromName, config.From)
		message += fmt.Sprintf("To: %s\r\n", config.To)
		message += fmt.Sprintf("Subject: %s\r\n", subject)
		message += "MIME-Version: 1.0\r\n"
		message += "Content-Type: text/html; charset=UTF-8\r\n"
		message += "\r\n"
		message += body
		return message
	}

	if isSSLPort {
		tlsConfig := &tls.Config{ServerName: config.Host}
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("failed to connect via SSL: %w", err)
		}
		defer conn.Close()

		client, err := smtp.NewClient(conn, config.Host)
		if err != nil {
			return fmt.Errorf("failed to create SMTP client: %w", err)
		}
		defer client.Close()

		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}
		if err = client.Mail(config.From); err != nil {
			return fmt.Errorf("failed to set sender: %w", err)
		}
		if err = client.Rcpt(config.To); err != nil {
			return fmt.Errorf("failed to set recipient: %w", err)
		}

		writer, err := client.Data()
		if err != nil {
			return fmt.Errorf("failed to get data writer: %w", err)
		}
		if _, err = writer.Write([]byte(buildMessage())); err != nil {
			return fmt.Errorf("failed to write message: %w", err)
		}
		if err = writer.Close(); err != nil {
			return fmt.Errorf("failed to close writer: %w", err)
		}
		return nil
	}

	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer client.Close()

	tlsConfig := &tls.Config{ServerName: config.Host}
	if err = client.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("failed to start TLS: %w", err)
	}
	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}
	if err = client.Mail(config.From); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}
	if err = client.Rcpt(config.To); err != nil {
		return fmt.Errorf("failed to set recipient: %w", err)
	}

	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}
	if _, err = writer.Write([]byte(buildMessage())); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}
	if err = writer.Close(); err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}

	return nil
}

func renderNotificationEmail(title string, intro string, subscription *models.Subscription, settings *SettingsService, dateLabel string, dateValue string) (string, error) {
	tmpl := `
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<style>
		body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
		.container { max-width: 600px; margin: 0 auto; padding: 20px; }
		.notice { background-color: #eef2ff; border: 1px solid #c7d2fe; border-radius: 8px; padding: 15px; margin: 20px 0; }
		.subscription-details { background-color: #f8f9fa; padding: 15px; border-radius: 8px; margin: 20px 0; }
		.detail-row { margin: 10px 0; }
		.label { font-weight: bold; }
		.footer { margin-top: 30px; padding-top: 20px; border-top: 1px solid #ddd; font-size: 12px; color: #666; }
	</style>
</head>
<body>
	<div class="container">
		<h2>{{.Title}}</h2>
		<div class="notice">
			<strong>提醒：</strong> {{.Intro}}
		</div>
		<div class="subscription-details">
			<h3>订阅信息</h3>
			<div class="detail-row"><span class="label">名称：</span>{{.Subscription.Name}}</div>
			<div class="detail-row"><span class="label">费用：</span>{{.CurrencySymbol}}{{printf "%.2f" .Subscription.Cost}} {{.Subscription.Schedule}}</div>
			<div class="detail-row"><span class="label">折算月费：</span>{{.CurrencySymbol}}{{printf "%.2f" (.Subscription.MonthlyCost)}}</div>
			{{if and .Subscription.Category .Subscription.Category.Name}}<div class="detail-row"><span class="label">分类：</span>{{.Subscription.Category.Name}}</div>{{end}}
			{{if .Subscription.PaymentMethod}}<div class="detail-row"><span class="label">支付方式：</span>{{.Subscription.PaymentMethod}}</div>{{end}}
			{{if .Subscription.Notes}}<div class="detail-row" style="white-space: pre-wrap"><span class="label">备注：</span>{{.Subscription.Notes}}</div>{{end}}
			{{if .DateValue}}<div class="detail-row"><span class="label">{{.DateLabel}}：</span>{{.DateValue}}</div>{{end}}
			{{if .Subscription.URL}}<div class="detail-row"><span class="label">链接：</span><a href="{{.Subscription.URL}}">{{.Subscription.URL}}</a></div>{{end}}
		</div>
		<div class="footer">
			<p>这是一封由 yyTrackr 自动发送的提醒邮件。</p>
			<p>你可以在设置页调整通知偏好。</p>
		</div>
	</div>
</body>
</html>
`

	data := struct {
		Title          string
		Intro          string
		Subscription   *models.Subscription
		CurrencySymbol string
		DateLabel      string
		DateValue      string
	}{
		Title:          title,
		Intro:          intro,
		Subscription:   subscription,
		CurrencySymbol: currencySymbolForSubscription(subscription, settings),
		DateLabel:      dateLabel,
		DateValue:      dateValue,
	}

	t, err := template.New("notificationEmail").Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("failed to parse email template: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute email template: %w", err)
	}

	return buf.String(), nil
}

func (e *EmailService) SendHighCostAlert(subscription *models.Subscription) error {
	enabled, err := e.settingsService.GetBoolSetting("high_cost_alerts", true)
	if err != nil || !enabled {
		return nil
	}

	var renewalDate string
	if subscription.RenewalDate != nil {
		renewalDate = subscription.RenewalDate.Format(e.settingsService.GetGoDateFormatLong())
	}

	body, err := renderNotificationEmail(
		"高价订阅提醒",
		"检测到新的高价订阅，请尽快确认是否继续保留。",
		subscription,
		e.settingsService,
		"下次续费",
		renewalDate,
	)
	if err != nil {
		return err
	}

	subject := fmt.Sprintf("高价订阅提醒｜%s｜%s%.2f/月", subscription.Name, currencySymbolForSubscription(subscription, e.settingsService), subscription.MonthlyCost())
	return e.SendEmail(subject, body)
}

func (e *EmailService) SendRenewalReminder(subscription *models.Subscription, daysUntilRenewal int) error {
	enabled, err := e.settingsService.GetBoolSetting("renewal_reminders", false)
	if err != nil || !enabled {
		return nil
	}

	var renewalDate string
	if subscription.RenewalDate != nil {
		renewalDate = subscription.RenewalDate.Format(e.settingsService.GetGoDateFormatLong())
	}

	body, err := renderNotificationEmail(
		"续费提醒",
		fmt.Sprintf("你的订阅 %s 将于 %s 续费。", subscription.Name, formatReminderLead(daysUntilRenewal)),
		subscription,
		e.settingsService,
		"续费日期",
		renewalDate,
	)
	if err != nil {
		return err
	}

	subject := fmt.Sprintf("续费提醒｜%s｜%s续费", subscription.Name, formatReminderLead(daysUntilRenewal))
	return e.SendEmail(subject, body)
}

func (e *EmailService) SendCancellationReminder(subscription *models.Subscription, daysUntilCancellation int) error {
	enabled, err := e.settingsService.GetBoolSetting("cancellation_reminders", false)
	if err != nil || !enabled {
		return nil
	}

	var cancellationDate string
	if subscription.CancellationDate != nil {
		cancellationDate = subscription.CancellationDate.Format(e.settingsService.GetGoDateFormatLong())
	}

	body, err := renderNotificationEmail(
		"到期提醒",
		fmt.Sprintf("你的订阅 %s 将于 %s 到期。", subscription.Name, formatReminderLead(daysUntilCancellation)),
		subscription,
		e.settingsService,
		"到期日期",
		cancellationDate,
	)
	if err != nil {
		return err
	}

	subject := fmt.Sprintf("到期提醒｜%s｜%s到期", subscription.Name, formatReminderLead(daysUntilCancellation))
	return e.SendEmail(subject, body)
}

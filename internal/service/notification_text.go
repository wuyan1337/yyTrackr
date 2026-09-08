package service

import (
	"fmt"
	"strings"
	"subtrackr/internal/models"
)

func formatReminderLead(days int) string {
	switch {
	case days <= 0:
		return "今天"
	case days == 1:
		return "明天"
	default:
		return fmt.Sprintf("%d 天后", days)
	}
}

func formatReminderDetail(days int, futureLabel string) string {
	switch {
	case days <= 0:
		return fmt.Sprintf("将于今天%s", futureLabel)
	case days == 1:
		return fmt.Sprintf("将于明天%s", futureLabel)
	default:
		return fmt.Sprintf("将于 %d 天后%s", days, futureLabel)
	}
}

func buildNotificationLines(subscription *models.Subscription, settings *SettingsService) []string {
	currencySymbol := currencySymbolForSubscription(subscription, settings)
	lines := []string{
		fmt.Sprintf("订阅名称：%s", subscription.Name),
		fmt.Sprintf("费用：%s%.2f", currencySymbol, subscription.Cost),
		fmt.Sprintf("计费周期：%s", subscription.Schedule),
		fmt.Sprintf("折算月费：%s%.2f", currencySymbol, subscription.MonthlyCost()),
	}

	if subscription.Category.Name != "" {
		lines = append(lines, fmt.Sprintf("分类：%s", subscription.Category.Name))
	}
	if subscription.URL != "" {
		lines = append(lines, fmt.Sprintf("链接：%s", subscription.URL))
	}
	if subscription.PaymentMethod != "" {
		lines = append(lines, fmt.Sprintf("支付方式：%s", subscription.PaymentMethod))
	}
	if subscription.Notes != "" {
		lines = append(lines, fmt.Sprintf("备注：%s", subscription.Notes))
	}

	return lines
}

func joinNotificationLines(lines ...string) string {
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			filtered = append(filtered, line)
		}
	}
	return strings.Join(filtered, "\n")
}

func currencySymbolForSubscription(subscription *models.Subscription, settings *SettingsService) string {
	preferred := settings.GetCurrency()
	if subscription.OriginalCurrency != "" && subscription.OriginalCurrency != preferred {
		return CurrencySymbolForCode(subscription.OriginalCurrency)
	}
	return settings.GetCurrencySymbol()
}

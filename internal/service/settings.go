package service

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"subtrackr/internal/models"
	"subtrackr/internal/repository"
)

type SettingsService struct {
	repo   *repository.SettingsRepository
	userID uint
}

var DefaultUIPersonalizationConfig = models.UIPersonalizationConfig{
	EnableChibiStickers: true,
}

func NewSettingsService(repo *repository.SettingsRepository) *SettingsService {
	return &SettingsService{repo: repo}
}

func (s *SettingsService) ForUser(userID uint) *SettingsService {
	return &SettingsService{
		repo:   s.repo,
		userID: userID,
	}
}

func (s *SettingsService) UserID() uint {
	return s.userID
}

func (s *SettingsService) set(key, value string) error {
	return s.repo.SetForUser(s.userID, key, value)
}

func (s *SettingsService) get(key string) (string, error) {
	return s.repo.GetForUser(s.userID, key)
}

func (s *SettingsService) del(key string) error {
	return s.repo.DeleteForUser(s.userID, key)
}

func (s *SettingsService) SetGlobal(key, value string) error {
	return s.repo.SetGlobal(key, value)
}

func (s *SettingsService) GetGlobal(key string) (string, error) {
	return s.repo.GetGlobal(key)
}

func (s *SettingsService) DeleteGlobal(key string) error {
	return s.repo.DeleteGlobal(key)
}

func (s *SettingsService) SaveUIPersonalizationConfig(config *models.UIPersonalizationConfig) error {
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	return s.set("ui_personalization_config", string(data))
}

func (s *SettingsService) GetUIPersonalizationConfig() (*models.UIPersonalizationConfig, error) {
	data, err := s.get("ui_personalization_config")
	if err != nil {
		return &models.UIPersonalizationConfig{
			EnableChibiStickers: DefaultUIPersonalizationConfig.EnableChibiStickers,
		}, err
	}

	var config models.UIPersonalizationConfig
	if err := json.Unmarshal([]byte(data), &config); err != nil {
		return nil, err
	}

	if data != "" && !config.EnableChibiStickers {
		return &config, nil
	}
	config.EnableChibiStickers = true
	return &config, nil
}

func (s *SettingsService) SetBoolSetting(key string, value bool) error {
	return s.set(key, fmt.Sprintf("%t", value))
}

func (s *SettingsService) GetBoolSetting(key string, defaultValue bool) (bool, error) {
	value, err := s.get(key)
	if err != nil {
		return defaultValue, err
	}
	return value == "true", nil
}

func (s *SettingsService) GetBoolSettingWithDefault(key string, defaultValue bool) bool {
	value, err := s.GetBoolSetting(key, defaultValue)
	if err != nil {
		return defaultValue
	}
	return value
}

func (s *SettingsService) SetIntSetting(key string, value int) error {
	return s.set(key, strconv.Itoa(value))
}

func (s *SettingsService) GetIntSetting(key string, defaultValue int) (int, error) {
	value, err := s.get(key)
	if err != nil {
		return defaultValue, err
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue, err
	}
	return intValue, nil
}

func (s *SettingsService) GetIntSettingWithDefault(key string, defaultValue int) int {
	value, err := s.GetIntSetting(key, defaultValue)
	if err != nil {
		return defaultValue
	}
	return value
}

func (s *SettingsService) SetFloatSetting(key string, value float64) error {
	return s.set(key, fmt.Sprintf("%.2f", value))
}

func (s *SettingsService) GetFloatSetting(key string, defaultValue float64) (float64, error) {
	value, err := s.get(key)
	if err != nil {
		return defaultValue, err
	}
	floatValue, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return defaultValue, err
	}
	return floatValue, nil
}

func (s *SettingsService) GetFloatSettingWithDefault(key string, defaultValue float64) float64 {
	value, err := s.GetFloatSetting(key, defaultValue)
	if err != nil {
		return defaultValue
	}
	return value
}

func (s *SettingsService) GetTheme() (string, error) {
	theme, err := s.get("theme")
	if err != nil {
		return "default", err
	}
	return theme, nil
}

func (s *SettingsService) SetTheme(theme string) error {
	return s.set("theme", theme)
}

func (s *SettingsService) CreateAPIKey(name, key string) (*models.APIKey, error) {
	apiKey := &models.APIKey{
		UserID: s.userID,
		Name:   name,
		Key:    key,
	}
	return s.repo.CreateAPIKey(apiKey)
}

func (s *SettingsService) GetAllAPIKeys() ([]models.APIKey, error) {
	return s.repo.GetAllAPIKeysForUser(s.userID)
}

func (s *SettingsService) DeleteAPIKey(id uint) error {
	return s.repo.DeleteAPIKeyForUser(s.userID, id)
}

func (s *SettingsService) ValidateAPIKey(key string) (*models.APIKey, error) {
	apiKey, err := s.repo.GetAPIKeyByKey(key)
	if err != nil {
		return nil, err
	}
	if err := s.repo.UpdateAPIKeyUsage(apiKey.ID); err != nil {
		return nil, err
	}
	return apiKey, nil
}

func (s *SettingsService) SetCurrency(currency string) error {
	if _, ok := currencyInfoMap[currency]; !ok {
		return fmt.Errorf("invalid currency: %s", currency)
	}
	return s.set("currency", currency)
}

func (s *SettingsService) GetCurrency() string {
	currency, err := s.get("currency")
	if err != nil || currency == "" {
		return "USD"
	}
	return currency
}

func CurrencySymbolForCode(currency string) string {
	return GetCurrencyInfo(currency).Symbol
}

func (s *SettingsService) GetCurrencySymbol() string {
	return CurrencySymbolForCode(s.GetCurrency())
}

func (s *SettingsService) SetDateFormat(format string) error {
	switch format {
	case "MM/DD/YYYY", "DD/MM/YYYY", "YYYY-MM-DD":
		return s.set("date_format", format)
	default:
		return fmt.Errorf("invalid date format: %s", format)
	}
}

func (s *SettingsService) GetDateFormat() string {
	format, err := s.get("date_format")
	if err != nil || format == "" {
		return "MM/DD/YYYY"
	}
	return format
}

func (s *SettingsService) GetGoDateFormat() string {
	return DateFormatToGo(s.GetDateFormat())
}

func (s *SettingsService) GetGoDateFormatLong() string {
	return DateFormatToGoLong(s.GetDateFormat())
}

func DateFormatToGo(format string) string {
	switch format {
	case "DD/MM/YYYY":
		return "02/01/2006"
	case "YYYY-MM-DD":
		return "2006-01-02"
	default:
		return "01/02/2006"
	}
}

func DateFormatToGoLong(format string) string {
	switch format {
	case "DD/MM/YYYY":
		return "2 January 2006"
	case "YYYY-MM-DD":
		return "2006-01-02"
	default:
		return "January 2, 2006"
	}
}

func (s *SettingsService) SetDarkMode(enabled bool) error {
	return s.SetBoolSetting("dark_mode", enabled)
}

func (s *SettingsService) IsDarkModeEnabled() bool {
	return s.GetBoolSettingWithDefault("dark_mode", false)
}

func (s *SettingsService) GetOrGenerateSessionSecret() (string, error) {
	secret, err := s.repo.GetGlobal("auth_session_secret")
	if err == nil && secret != "" {
		return secret, nil
	}

	bytes := make([]byte, 64)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	secret = base64.URLEncoding.EncodeToString(bytes)

	if err := s.repo.SetGlobal("auth_session_secret", secret); err != nil {
		return "", err
	}

	return secret, nil
}

func (s *SettingsService) SaveTelegramConfig(config *models.TelegramConfig) error {
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	return s.set("telegram_config", string(data))
}

func (s *SettingsService) GetTelegramConfig() (*models.TelegramConfig, error) {
	data, err := s.get("telegram_config")
	if err != nil {
		return nil, err
	}

	var config models.TelegramConfig
	if err := json.Unmarshal([]byte(data), &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func (s *SettingsService) SaveWebhookConfig(config *models.WebhookConfig) error {
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	return s.set("webhook_config", string(data))
}

func (s *SettingsService) GetWebhookConfig() (*models.WebhookConfig, error) {
	data, err := s.get("webhook_config")
	if err != nil {
		return nil, err
	}
	var config models.WebhookConfig
	if err := json.Unmarshal([]byte(data), &config); err != nil {
		return nil, err
	}
	return &config, nil
}

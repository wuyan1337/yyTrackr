package service

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/url"
	"strings"
	"subtrackr/internal/models"
	"subtrackr/internal/repository"
	"time"
)

type CurrencyInfo struct {
	Code   string `json:"code"`
	Symbol string `json:"symbol"`
	Name   string `json:"name"`
}

var BuiltinCurrencies = []CurrencyInfo{
	{Code: "USD", Symbol: "$", Name: "US Dollar"},
	{Code: "EUR", Symbol: "€", Name: "Euro"},
	{Code: "GBP", Symbol: "£", Name: "British Pound"},
	{Code: "AUD", Symbol: "A$", Name: "Australian Dollar"},
	{Code: "CAD", Symbol: "C$", Name: "Canadian Dollar"},
	{Code: "NZD", Symbol: "NZ$", Name: "New Zealand Dollar"},
	{Code: "JPY", Symbol: "¥", Name: "Japanese Yen"},
	{Code: "CHF", Symbol: "Fr.", Name: "Swiss Franc"},
	{Code: "CNY", Symbol: "￥", Name: "Chinese Yuan"},
	{Code: "SEK", Symbol: "kr", Name: "Swedish Krona"},
	{Code: "NOK", Symbol: "kr", Name: "Norwegian Krone"},
	{Code: "DKK", Symbol: "kr", Name: "Danish Krone"},
	{Code: "INR", Symbol: "₹", Name: "Indian Rupee"},
	{Code: "RUB", Symbol: "₽", Name: "Russian Ruble"},
	{Code: "BRL", Symbol: "R$", Name: "Brazilian Real"},
	{Code: "PLN", Symbol: "zł", Name: "Polish Zloty"},
	{Code: "KRW", Symbol: "₩", Name: "South Korean Won"},
	{Code: "SGD", Symbol: "S$", Name: "Singapore Dollar"},
	{Code: "HKD", Symbol: "HK$", Name: "Hong Kong Dollar"},
	{Code: "MXN", Symbol: "Mex$", Name: "Mexican Peso"},
	{Code: "ZAR", Symbol: "R", Name: "South African Rand"},
	{Code: "TRY", Symbol: "₺", Name: "Turkish Lira"},
	{Code: "THB", Symbol: "฿", Name: "Thai Baht"},
	{Code: "COP", Symbol: "COL$", Name: "Colombian Peso"},
	{Code: "BDT", Symbol: "৳", Name: "Bangladeshi Taka"},
	{Code: "IDR", Symbol: "Rp", Name: "Indonesian Rupiah"},
	{Code: "PHP", Symbol: "₱", Name: "Philippine Peso"},
	{Code: "TWD", Symbol: "NT$", Name: "New Taiwan Dollar"},
	{Code: "MYR", Symbol: "RM", Name: "Malaysian Ringgit"},
	{Code: "AED", Symbol: "د.إ", Name: "UAE Dirham"},
	{Code: "SAR", Symbol: "﷼", Name: "Saudi Riyal"},
	{Code: "ILS", Symbol: "₪", Name: "Israeli Shekel"},
	{Code: "CZK", Symbol: "Kč", Name: "Czech Koruna"},
	{Code: "HUF", Symbol: "Ft", Name: "Hungarian Forint"},
	{Code: "RON", Symbol: "lei", Name: "Romanian Leu"},
}

var currencyInfoMap map[string]CurrencyInfo
var SupportedCurrencies []string

func init() {
	currencyInfoMap = make(map[string]CurrencyInfo, len(BuiltinCurrencies))
	SupportedCurrencies = make([]string, len(BuiltinCurrencies))
	for i, c := range BuiltinCurrencies {
		currencyInfoMap[c.Code] = c
		SupportedCurrencies[i] = c.Code
	}
}

func GetCurrencyInfo(code string) CurrencyInfo {
	if info, ok := currencyInfoMap[code]; ok {
		return info
	}
	return CurrencyInfo{Code: code, Symbol: code, Name: code}
}

func GetAvailableCurrencies() []CurrencyInfo {
	return BuiltinCurrencies
}

func supportedCurrencySymbols() string {
	return strings.Join(SupportedCurrencies, ",")
}

type CurrencyService struct {
	repo    *repository.ExchangeRateRepository
	baseURL string
	client  *http.Client
}

type frankfurterRate struct {
	Date  string  `json:"date"`
	Base  string  `json:"base"`
	Quote string  `json:"quote"`
	Rate  float64 `json:"rate"`
}

func NewCurrencyService(repo *repository.ExchangeRateRepository) *CurrencyService {
	return &CurrencyService{
		repo:    repo,
		baseURL: "https://api.frankfurter.dev/v2",
	}
}

func (s *CurrencyService) IsEnabled() bool {
	return true
}

func (s *CurrencyService) GetExchangeRate(fromCurrency, toCurrency string) (float64, error) {
	for _, code := range []string{fromCurrency, toCurrency} {
		if _, ok := currencyInfoMap[code]; !ok {
			return 0, fmt.Errorf("unsupported currency: %s", code)
		}
	}
	if fromCurrency == toCurrency {
		return 1.0, nil
	}

	rate, err := s.repo.GetRate(fromCurrency, toCurrency)
	// Cache freshness is fetch time, not the provider's business-day date.
	if err == nil && rate.Rate > 0 && !math.IsNaN(rate.Rate) && !math.IsInf(rate.Rate, 0) && time.Since(rate.CreatedAt) >= 0 && time.Since(rate.CreatedAt) < 24*time.Hour {
		return rate.Rate, nil
	}

	return s.fetchAndCacheRate(fromCurrency, toCurrency)
}

func (s *CurrencyService) ConvertAmount(amount float64, fromCurrency, toCurrency string) (float64, error) {
	rate, err := s.GetExchangeRate(fromCurrency, toCurrency)
	if err != nil {
		return 0, err
	}
	return amount * rate, nil
}

func (s *CurrencyService) fetchAndCacheRate(baseCurrency, targetCurrency string) (float64, error) {
	endpoint := fmt.Sprintf("%s/rates?base=%s&quotes=%s", s.baseURL, url.QueryEscape(baseCurrency), url.QueryEscape(targetCurrency))

	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return 0, fmt.Errorf("invalid API URL: %w", err)
	}
	if parsedURL.Host != "api.frankfurter.dev" {
		return 0, fmt.Errorf("unauthorized API host: %s", parsedURL.Host)
	}

	client := s.client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second, Transport: &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12}}}
	}

	resp, err := client.Get(endpoint)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch exchange rate: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("exchange rate API returned status %d", resp.StatusCode)
	}

	var payload []frankfurterRate
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return 0, fmt.Errorf("failed to decode exchange rate response: %w", err)
	}
	if len(payload) == 0 {
		return 0, fmt.Errorf("exchange rate for %s to %s not available", baseCurrency, targetCurrency)
	}

	item := payload[0]
	if item.Rate <= 0 || math.IsNaN(item.Rate) || math.IsInf(item.Rate, 0) || item.Base != baseCurrency || item.Quote != targetCurrency {
		return 0, fmt.Errorf("exchange rate for %s to %s not available", baseCurrency, targetCurrency)
	}

	rateDate := time.Now().UTC()
	if item.Date != "" {
		if parsedDate, err := time.Parse("2006-01-02", item.Date); err == nil {
			rateDate = parsedDate
		}
	}

	if err := s.repo.SaveRates([]models.ExchangeRate{
		{
			BaseCurrency: strings.ToUpper(baseCurrency),
			Currency:     strings.ToUpper(targetCurrency),
			Rate:         item.Rate,
			Date:         rateDate,
		},
	}); err != nil {
		log.Printf("Warning: failed to cache exchange rate %s -> %s: %v", baseCurrency, targetCurrency, err)
	}

	return item.Rate, nil
}

func (s *CurrencyService) RefreshRates() error {
	baseCurrency := "USD"
	endpoint := fmt.Sprintf("%s/rates?base=%s&quotes=%s", s.baseURL, url.QueryEscape(baseCurrency), url.QueryEscape(supportedCurrencySymbols()))

	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("invalid API URL: %w", err)
	}
	if parsedURL.Host != "api.frankfurter.dev" {
		return fmt.Errorf("unauthorized API host: %s", parsedURL.Host)
	}

	client := s.client
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second, Transport: &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12}}}
	}

	resp, err := client.Get(endpoint)
	if err != nil {
		return fmt.Errorf("failed to refresh exchange rates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("exchange rate API returned status %d", resp.StatusCode)
	}

	var payload []frankfurterRate
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return fmt.Errorf("failed to decode exchange rate response: %w", err)
	}

	rateDate := time.Now().UTC()
	ratesToSave := make([]models.ExchangeRate, 0, len(payload)+1)
	ratesToSave = append(ratesToSave, models.ExchangeRate{
		BaseCurrency: baseCurrency,
		Currency:     baseCurrency,
		Rate:         1.0,
		Date:         rateDate,
	})

	for _, item := range payload {
		if item.Rate <= 0 || math.IsNaN(item.Rate) || math.IsInf(item.Rate, 0) || item.Quote == "" || item.Base != baseCurrency {
			continue
		}
		rateDate = time.Now().UTC()
		if item.Date != "" {
			if parsedDate, err := time.Parse("2006-01-02", item.Date); err == nil {
				rateDate = parsedDate
			}
		}
		ratesToSave = append(ratesToSave, models.ExchangeRate{
			BaseCurrency: strings.ToUpper(baseCurrency),
			Currency:     strings.ToUpper(item.Quote),
			Rate:         item.Rate,
			Date:         rateDate,
		})
	}

	if len(ratesToSave) == 1 {
		return fmt.Errorf("no valid exchange rates available")
	}
	if err := s.repo.SaveRates(ratesToSave); err != nil {
		return fmt.Errorf("failed to cache exchange rates: %w", err)
	}

	return s.repo.DeleteStaleRates(7 * 24 * time.Hour)
}

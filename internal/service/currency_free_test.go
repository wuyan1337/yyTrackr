package service

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"subtrackr/internal/models"
	"subtrackr/internal/repository"
	"testing"
	"time"
)

type currencyTransport func(*http.Request) (*http.Response, error)

func (f currencyTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func mockCurrency(t *testing.T, status int, body string) (*CurrencyService, *int) {
	t.Helper()
	s := NewCurrencyService(repository.NewExchangeRateRepository(setupTestDB(t)))
	calls := new(int)
	s.client = &http.Client{Timeout: time.Second, Transport: currencyTransport(func(r *http.Request) (*http.Response, error) {
		*calls = *calls + 1
		if r.URL.Host != "api.frankfurter.dev" || r.URL.Query().Get("access_key") != "" || r.Header.Get("Authorization") != "" {
			t.Errorf("unexpected request: %s", r.URL)
		}
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}
	return s, calls
}
func TestCurrencyRejectsUnsupportedSameCurrency(t *testing.T) {
	s, _ := mockCurrency(t, 200, "[]")
	amount, err := s.ConvertAmount(100, "XYZ", "XYZ")
	if err == nil || amount != 0 {
		t.Fatalf("unsupported currency must fail, got %v, %v", amount, err)
	}
}
func TestCurrencyFreeFetchAndCache(t *testing.T) {
	t.Setenv("FIXER_API_KEY", "")
	s, calls := mockCurrency(t, 200, `[{"date":"2020-01-03","base":"USD","quote":"EUR","rate":0.8}]`)
	for i := 0; i < 2; i++ {
		value, err := s.ConvertAmount(100, "USD", "EUR")
		if err != nil || value != 80 {
			t.Fatalf("conversion: %v %v", value, err)
		}
	}
	if *calls != 1 {
		t.Fatalf("old provider date must still cache newly fetched response, requests=%d", *calls)
	}
	rate, err := s.repo.GetRate("USD", "EUR")
	if err != nil || rate.Date.Format("2006-01-02") != "2020-01-03" {
		t.Fatal("provider date lost", err)
	}
}
func TestCurrencyFreeFailures(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
	}{
		{"unavailable", 503, `{}`}, {"malformed", 200, `bad`}, {"empty", 200, `[]`}, {"zero", 200, `[{"base":"USD","quote":"EUR","rate":0}]`}, {"negative", 200, `[{"base":"USD","quote":"EUR","rate":-1}]`}, {"wrong_pair", 200, `[{"base":"GBP","quote":"EUR","rate":0.8}]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _ := mockCurrency(t, tc.status, tc.body)
			value, err := s.ConvertAmount(100, "USD", "EUR")
			if err == nil || value != 0 {
				t.Fatalf("must not invent a conversion: %v %v", value, err)
			}
		})
	}
}
func TestCurrencyFreeTimeoutAndSameCurrency(t *testing.T) {
	s, calls := mockCurrency(t, 200, `[]`)
	if v, e := s.ConvertAmount(100, "USD", "USD"); e != nil || v != 100 || *calls != 0 {
		t.Fatal(v, e, *calls)
	}
	s.client = &http.Client{Timeout: time.Millisecond, Transport: currencyTransport(func(r *http.Request) (*http.Response, error) { <-r.Context().Done(); return nil, r.Context().Err() })}
	if v, e := s.ConvertAmount(100, "USD", "EUR"); e == nil || v != 0 {
		t.Fatal(v, e)
	}
}
func TestCurrencyFreeRefreshValidation(t *testing.T) {
	for _, body := range []string{`[]`, `[{"base":"USD","quote":"EUR","rate":-1}]`, `[{"base":"USD","quote":"EUR","rate":0}]`, `[{"base":"GBP","quote":"EUR","rate":0.8}]`, `[{"base":"USD","quote":"EUR","rate":1e999}]`, `[{"base":"USD","quote":"EUR","rate":NaN}]`} {
		s, calls := mockCurrency(t, 200, body)
		if err := s.RefreshRates(); err == nil {
			t.Fatalf("invalid refresh accepted: %s", body)
		}
		if *calls != 1 {
			t.Fatal("refresh must use injected client")
		}
		if _, err := s.repo.GetRate("USD", "EUR"); err == nil {
			t.Fatal("invalid rate cached")
		}
		if v, err := s.GetExchangeRate("USD", "EUR"); err == nil || v != 0 {
			t.Fatalf("invalid fetch accepted: %v %v", v, err)
		}
	}
}
func TestCurrencyFreeWeekendRefresh(t *testing.T) {
	s, calls := mockCurrency(t, 200, `[{"date":"2020-01-03","base":"USD","quote":"EUR","rate":0.8}]`)
	old := time.Now().Add(-48 * time.Hour)
	if err := s.repo.SaveRates([]models.ExchangeRate{{BaseCurrency: "USD", Currency: "EUR", Rate: 0.5, Date: time.Date(2020, 1, 3, 0, 0, 0, 0, time.UTC), CreatedAt: old}}); err != nil {
		t.Fatal(err)
	}
	if err := s.RefreshRates(); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if v, e := s.GetExchangeRate("USD", "EUR"); e != nil || v != 0.8 {
			t.Fatal(v, e)
		}
	}
	if *calls != 1 {
		t.Fatal("same provider date must select latest fetch", *calls)
	}
}
func TestCurrencyFreeStaleCacheRefresh(t *testing.T) {
	s, calls := mockCurrency(t, 200, fmt.Sprintf(`[{"date":%q,"base":"USD","quote":"EUR","rate":0.8}]`, time.Now().Format("2006-01-02")))
	old := time.Now().Add(-48 * time.Hour)
	if err := s.repo.SaveRates([]models.ExchangeRate{{BaseCurrency: "USD", Currency: "EUR", Rate: 0.5, Date: old, CreatedAt: old}}); err != nil {
		t.Fatal(err)
	}
	if v, e := s.ConvertAmount(100, "USD", "EUR"); e != nil || v != 80 || *calls != 1 {
		t.Fatal(v, e, *calls)
	}
}

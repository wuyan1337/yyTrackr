package handlers

import (
	"bytes"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"html/template"
	"net/http/httptest"
	"os"
	"strings"
	"subtrackr/internal/models"
	"subtrackr/internal/repository"
	"subtrackr/internal/service"
	"testing"
)

func TestSettingsCleanupRegression(t *testing.T) {
	raw, err := os.ReadFile("../../templates/settings.html")
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, bad := range []string{"基础地址", "iCal 订阅", "数据管理", "/api/backup", "/api/clear-all", "邮件通知", "smtp-form", "pushover-form", "当前站点采用强制登录模式", "已启用强制登录与账户隔离"} {
		if strings.Contains(text, bad) {
			t.Errorf("still contains %q", bad)
		}
	}
	for _, want := range []string{"/api/auth/logout", "退出登录", "telegram-form", "webhook-form", "renewal-toggle", "cancellation-toggle"} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q", want)
		}
	}
	if _, err := template.New("settings.html").Parse(text); err != nil {
		t.Fatal(err)
	}
}

func TestFormCurrencyRegression(t *testing.T) {
	tmpl, err := template.ParseFiles("../../templates/subscription-form.html")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, defaultCode, want string
		sub                     *models.Subscription
	}{
		{"new CNY", "CNY", "CNY", nil}, {"new EUR", "EUR", "EUR", nil}, {"edit USD", "CNY", "USD", &models.Subscription{OriginalCurrency: "USD"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var b bytes.Buffer
			err := tmpl.Execute(&b, map[string]interface{}{"DefaultCurrency": tc.defaultCode, "Subscription": tc.sub, "Currencies": service.GetAvailableCurrencies()})
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(b.String(), `value="`+tc.want+`" selected`) {
				t.Errorf("wrong selected currency: want %s", tc.want)
			}
			if strings.Contains(b.String(), "absolute left-3 top-2") {
				t.Error("cost prefix remains")
			}
		})
	}
}

func TestLogoutRegression(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	db.AutoMigrate(&models.User{})
	users := service.NewUserService(repository.NewUserRepository(db))
	user, err := users.Create("logout-test", "test-password-only")
	if err != nil {
		t.Fatal(err)
	}
	sessions := service.NewSessionService("isolated-test-secret")
	req := httptest.NewRequest("GET", "/", nil)
	logged := httptest.NewRecorder()
	if err := sessions.CreateSession(logged, req, user.ID, false); err != nil {
		t.Fatal(err)
	}
	req = httptest.NewRequest("GET", "/api/auth/logout", nil)
	req.AddCookie(logged.Result().Cookies()[0])
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	NewAuthHandler(users, nil, sessions).Logout(c)
	if w.Code != 302 || w.Header().Get("Location") != "/login" {
		t.Fatalf("logout response %d %s", w.Code, w.Body.String())
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || cookies[0].MaxAge != -1 {
		t.Fatalf("cookie not cleared: %v", cookies)
	}
	next := httptest.NewRequest("GET", "/settings", nil)
	next.AddCookie(cookies[0])
	if _, ok := sessions.GetCurrentUserID(next); ok {
		t.Fatal("still authenticated")
	}
}

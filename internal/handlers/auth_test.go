package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"subtrackr/internal/models"
	"subtrackr/internal/repository"
	"subtrackr/internal/service"
	"testing"
)

func TestLoginRememberMeCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	if err := db.AutoMigrate(&models.User{}); err != nil {
		t.Fatal(err)
	}
	users := service.NewUserService(repository.NewUserRepository(db))
	user, err := users.Create("remember-test", "test-password-only")
	if err != nil {
		t.Fatal(err)
	}
	sessions := service.NewSessionService("isolated-test-secret-not-production")
	h := NewAuthHandler(users, nil, sessions)
	for _, tc := range []struct {
		name, value string
		age         int
	}{
		{"browser_checkbox", "on", service.RememberMeMaxAge},
		{"boolean_true", "true", service.RememberMeMaxAge},
		{"numeric_true", "1", service.RememberMeMaxAge},
		{"unchecked", "", service.SessionMaxAge},
		{"explicit_false", "false", service.SessionMaxAge},
		{"invalid", "invalid", service.SessionMaxAge},
	} {
		t.Run(tc.name, func(t *testing.T) {
			form := url.Values{"username": {"remember-test"}, "password": {"test-password-only"}}
			if tc.value != "" {
				form.Set("remember_me", tc.value)
			}
			req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req
			h.Login(c)
			if w.Code != http.StatusOK || w.Header().Get("HX-Redirect") != "/" {
				t.Fatalf("login failed: %d %s", w.Code, w.Body.String())
			}
			cookies := w.Result().Cookies()
			if len(cookies) != 1 {
				t.Fatalf("cookies: %v", cookies)
			}
			cookie := cookies[0]
			if cookie.MaxAge != tc.age {
				t.Errorf("MaxAge = %d, want %d", cookie.MaxAge, tc.age)
			}
			if !cookie.HttpOnly || cookie.Path != "/" {
				t.Errorf("invalid cookie attributes: %v", cookie)
			}
			next := httptest.NewRequest(http.MethodGet, "/", nil)
			next.AddCookie(cookie)
			restarted := service.NewSessionService("isolated-test-secret-not-production")
			if id, ok := restarted.GetCurrentUserID(next); !ok || id != user.ID {
				t.Fatal("session lost after service recreation")
			}
		})
	}
}

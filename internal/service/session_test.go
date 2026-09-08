package service

import (
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"net/http/httptest"
	"subtrackr/internal/models"
	"subtrackr/internal/repository"
	"testing"
)

func TestRememberMeSecretSurvivesRestart(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	if err := db.AutoMigrate(&models.Settings{}); err != nil {
		t.Fatal(err)
	}
	settings := NewSettingsService(repository.NewSettingsRepository(db))
	secret, err := settings.GetOrGenerateSessionSecret()
	if err != nil {
		t.Fatal(err)
	}
	first := NewSessionService(secret)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/login", nil)
	if err := first.CreateSession(w, req, 42, true); err != nil {
		t.Fatal(err)
	}
	restartedSettings := NewSettingsService(repository.NewSettingsRepository(db))
	saved, err := restartedSettings.GetOrGenerateSessionSecret()
	if err != nil {
		t.Fatal(err)
	}
	if saved != secret {
		t.Fatal("persisted signing secret changed")
	}
	cookie := w.Result().Cookies()[0]
	if cookie.MaxAge != RememberMeMaxAge {
		t.Fatalf("MaxAge = %d", cookie.MaxAge)
	}
	next := httptest.NewRequest("GET", "/", nil)
	next.AddCookie(cookie)
	restarted := NewSessionService(saved)
	if id, ok := restarted.GetCurrentUserID(next); !ok || id != 42 {
		t.Fatal("remembered session lost on restart")
	}
	// A different signing secret must not authenticate the same cookie.
	if NewSessionService("different-secret").IsAuthenticated(next) {
		t.Fatal("accepted cookie signed by another key")
	}
}

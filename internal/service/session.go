package service

import (
	"net/http"

	"github.com/gorilla/securecookie"
)

const (
	SessionName      = "subtrackr_session"
	SessionUserKey   = "user_authenticated"
	SessionUserIDKey = "user_id"
	SessionMaxAge    = 24 * 60 * 60      // 24 hours in seconds
	RememberMeMaxAge = 30 * 24 * 60 * 60 // 30 days in seconds
)

type sessionPayload struct {
	Authenticated bool `json:"authenticated"`
	UserID        uint `json:"user_id"`
}

type SessionService struct {
	codec *securecookie.SecureCookie
}

func NewSessionService(secretKey string) *SessionService {
	codec := securecookie.New([]byte(secretKey), nil)
	codec.MaxAge(RememberMeMaxAge)

	return &SessionService{codec: codec}
}

func (s *SessionService) CreateSession(w http.ResponseWriter, r *http.Request, userID uint, rememberMe bool) error {
	maxAge := SessionMaxAge
	if rememberMe {
		maxAge = RememberMeMaxAge
	}

	encoded, err := s.codec.Encode(SessionName, sessionPayload{
		Authenticated: true,
		UserID:        userID,
	})
	if err != nil {
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     SessionName,
		Value:    encoded,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteStrictMode,
	})

	return nil
}

func (s *SessionService) readPayload(r *http.Request) (*sessionPayload, bool) {
	cookie, err := r.Cookie(SessionName)
	if err != nil || cookie.Value == "" {
		return nil, false
	}

	var payload sessionPayload
	if err := s.codec.Decode(SessionName, cookie.Value, &payload); err != nil {
		return nil, false
	}

	if !payload.Authenticated || payload.UserID == 0 {
		return nil, false
	}

	return &payload, true
}

func (s *SessionService) IsAuthenticated(r *http.Request) bool {
	_, ok := s.readPayload(r)
	return ok
}

func (s *SessionService) GetCurrentUserID(r *http.Request) (uint, bool) {
	payload, ok := s.readPayload(r)
	if !ok {
		return 0, false
	}
	return payload.UserID, true
}

func (s *SessionService) DestroySession(w http.ResponseWriter, r *http.Request) error {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteStrictMode,
	})
	return nil
}

func (s *SessionService) RefreshSession(w http.ResponseWriter, r *http.Request) error {
	payload, ok := s.readPayload(r)
	if !ok {
		return nil
	}
	return s.CreateSession(w, r, payload.UserID, false)
}

func (s *SessionService) UpdateSessionExpiry(maxAge int) {
	s.codec.MaxAge(maxAge)
}

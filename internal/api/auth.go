package api

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/ncecere/chaptarr/internal/model"
	"github.com/ncecere/chaptarr/internal/service"
	"github.com/ncecere/chaptarr/internal/store"
)

const (
	sessionCookie = "chaptarr_session"
	sessionTTL    = 30 * 24 * time.Hour
	minPassword   = 8
)

func randomToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

// authenticate resolves the caller from the session cookie or an API key
// (X-Api-Key header, or ?apikey= for tools that can only build URLs).
func (s *Server) authenticate(r *http.Request) (*store.UserRecord, string) {
	if c, err := r.Cookie(sessionCookie); err == nil && c.Value != "" {
		if u, err := s.store.UserBySession(c.Value); err == nil {
			return u, c.Value
		}
	}
	key := r.Header.Get("X-Api-Key")
	if key == "" {
		key = r.URL.Query().Get("apikey")
	}
	if key != "" {
		if u, err := s.store.UserByAPIKey(key); err == nil && subtle.ConstantTimeCompare([]byte(u.APIKey), []byte(key)) == 1 {
			return u, ""
		}
	}
	return nil, ""
}

func isHTTPS(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

func (s *Server) startSession(w http.ResponseWriter, r *http.Request, u *store.UserRecord) error {
	token := randomToken()
	expires := time.Now().Add(sessionTTL)
	if err := s.store.CreateSession(token, u.ID, expires); err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: token, Path: "/", Expires: expires,
		HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: isHTTPS(r),
	})
	return nil
}

func (s *Server) statusFor(u *store.UserRecord) (*model.AuthStatus, error) {
	n, err := s.store.UserCount()
	if err != nil {
		return nil, err
	}
	st := &model.AuthStatus{SetupRequired: n == 0}
	if u != nil {
		st.Authenticated = true
		st.User = &u.User
	}
	return st, nil
}

func (s *Server) authStatus(w http.ResponseWriter, r *http.Request) (any, error) {
	return s.statusFor(currentUser(r))
}

type credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *Server) setup(w http.ResponseWriter, r *http.Request) (any, error) {
	var c credentials
	if err := decode(r, &c); err != nil {
		return nil, err
	}
	c.Username = strings.TrimSpace(c.Username)
	if c.Username == "" {
		return nil, badRequest("Choose a username")
	}
	if len(c.Password) < minPassword {
		return nil, badRequest("The password must be at least 8 characters")
	}
	if len(c.Password) > 72 {
		return nil, badRequest("The password must be at most 72 characters")
	}
	// Serialised so two simultaneous requests cannot both create "the first" account.
	s.setupMu.Lock()
	defer s.setupMu.Unlock()
	n, err := s.store.UserCount()
	if err != nil {
		return nil, err
	}
	if n > 0 {
		return nil, &service.Error{Status: http.StatusConflict, Msg: "Setup is already complete. Sign in instead."}
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(c.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	u, err := s.store.CreateUser(c.Username, string(hash), randomToken())
	if err != nil {
		return nil, err
	}
	if err := s.startSession(w, r, u); err != nil {
		return nil, err
	}
	s.log.Info("admin account created", "username", u.Username)
	return s.statusFor(u)
}

// A fixed hash to compare against when the username is unknown, so the
// response time does not reveal which usernames exist.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("chaptarr-dummy-password"), bcrypt.DefaultCost)

func (s *Server) login(w http.ResponseWriter, r *http.Request) (any, error) {
	ip := clientIP(r)
	if wait := s.limiter.blocked(ip); wait > 0 {
		return nil, &service.Error{Status: http.StatusTooManyRequests, Msg: "Too many failed sign-in attempts. Try again in a few minutes."}
	}
	var c credentials
	if err := decode(r, &c); err != nil {
		return nil, err
	}
	u, err := s.store.UserByName(strings.TrimSpace(c.Username))
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return nil, err
	}
	hash := dummyHash
	if u != nil {
		hash = []byte(u.PasswordHash)
	}
	if bcrypt.CompareHashAndPassword(hash, []byte(c.Password)) != nil || u == nil {
		s.limiter.fail(ip)
		s.log.Warn("failed sign-in", "username", c.Username, "ip", ip)
		return nil, &service.Error{Status: http.StatusUnauthorized, Msg: "Incorrect username or password"}
	}
	s.limiter.reset(ip)
	_ = s.store.PurgeExpiredSessions()
	if err := s.startSession(w, r, u); err != nil {
		return nil, err
	}
	return s.statusFor(u)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) (any, error) {
	if token, _ := r.Context().Value(sessionKey).(string); token != "" {
		if err := s.store.DeleteSession(token); err != nil {
			return nil, err
		}
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	return nil, nil
}

func (s *Server) changePassword(w http.ResponseWriter, r *http.Request) (any, error) {
	var req struct {
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
	}
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	u := currentUser(r)
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.CurrentPassword)) != nil {
		return nil, badRequest("The current password is incorrect")
	}
	if len(req.NewPassword) < minPassword {
		return nil, badRequest("The new password must be at least 8 characters")
	}
	if len(req.NewPassword) > 72 {
		return nil, badRequest("The new password must be at most 72 characters")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	if err := s.store.SetPassword(u.ID, string(hash)); err != nil {
		return nil, err
	}
	// Sign out every other device; this one stays signed in.
	token, _ := r.Context().Value(sessionKey).(string)
	return nil, s.store.DeleteUserSessions(u.ID, token)
}

func (s *Server) regenerateAPIKey(w http.ResponseWriter, r *http.Request) (any, error) {
	key := randomToken()
	if err := s.store.SetAPIKey(currentUser(r).ID, key); err != nil {
		return nil, err
	}
	return map[string]string{"apiKey": key}, nil
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// loginLimiter locks an address out for a while after repeated failures.
type loginLimiter struct {
	mu      sync.Mutex
	entries map[string]*loginEntry
}

type loginEntry struct {
	failures int
	first    time.Time
	until    time.Time
}

const (
	maxFailures   = 8
	failureWindow = 10 * time.Minute
	lockout       = 5 * time.Minute
)

func (l *loginLimiter) blocked(ip string) time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()
	if e := l.entries[ip]; e != nil {
		return time.Until(e.until)
	}
	return 0
}

func (l *loginLimiter) fail(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.entries == nil {
		l.entries = map[string]*loginEntry{}
	}
	now := time.Now()
	// Forget stale entries so the map cannot grow without bound.
	for k, e := range l.entries {
		if now.Sub(e.first) > failureWindow && now.After(e.until) {
			delete(l.entries, k)
		}
	}
	e := l.entries[ip]
	if e == nil {
		e = &loginEntry{first: now}
		l.entries[ip] = e
	}
	e.failures++
	if e.failures >= maxFailures {
		e.until = now.Add(lockout)
		e.failures = 0
		e.first = now
	}
}

func (l *loginLimiter) reset(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.entries, ip)
}

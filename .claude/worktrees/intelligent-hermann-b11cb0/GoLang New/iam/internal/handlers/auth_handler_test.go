package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/mainframe/tm-system/iam/internal/handlers"
	"github.com/mainframe/tm-system/iam/internal/middleware"
	"github.com/mainframe/tm-system/iam/internal/models"
	"github.com/mainframe/tm-system/iam/internal/repository"
	"github.com/mainframe/tm-system/iam/internal/service"
)

// ── minimal mock stores ───────────────────────────────────────────────────────

type memUserStore struct {
	byName map[string]*models.User
	byID   map[string]*models.User
}

func newMemUserStore(users ...*models.User) *memUserStore {
	m := &memUserStore{
		byName: make(map[string]*models.User),
		byID:   make(map[string]*models.User),
	}
	for _, u := range users {
		m.byName[u.Username] = u
		m.byID[u.ID] = u
	}
	return m
}

func (m *memUserStore) FindByUsername(_ context.Context, username string) (*models.User, error) {
	u, ok := m.byName[username]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return u, nil
}
func (m *memUserStore) FindByID(_ context.Context, id string) (*models.User, error) {
	u, ok := m.byID[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return u, nil
}
func (m *memUserStore) Create(_ context.Context, u *models.User) error { return nil }
func (m *memUserStore) List(_ context.Context) ([]models.User, error)  { return nil, nil }
func (m *memUserStore) Update(_ context.Context, id string, fields map[string]interface{}) error {
	u, ok := m.byID[id]
	if !ok {
		return repository.ErrNotFound
	}
	if pw, ok := fields["password_hash"]; ok {
		u.PasswordHash = pw.(string)
	}
	return nil
}
func (m *memUserStore) Delete(_ context.Context, id string) error          { return nil }
func (m *memUserStore) Count(_ context.Context) (int64, error)             { return 0, nil }
func (m *memUserStore) UpdateLastLogin(_ context.Context, id string) error { return nil }

type memTokenStore struct {
	tokens map[string]*models.RefreshToken
}

func newMemTokenStore() *memTokenStore {
	return &memTokenStore{tokens: make(map[string]*models.RefreshToken)}
}

func (m *memTokenStore) Create(_ context.Context, t *models.RefreshToken) error {
	m.tokens[t.Token] = t
	return nil
}
func (m *memTokenStore) FindByToken(_ context.Context, token string) (*models.RefreshToken, error) {
	t, ok := m.tokens[token]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return t, nil
}
func (m *memTokenStore) Revoke(_ context.Context, token string) error {
	if t, ok := m.tokens[token]; ok {
		t.Revoked = true
	}
	return nil
}
func (m *memTokenStore) RevokeAllForUser(_ context.Context, userID string) error { return nil }
func (m *memTokenStore) DeleteExpired(_ context.Context) error                   { return nil }

// ── helpers ───────────────────────────────────────────────────────────────────

const handlerSecret = "handler-test-jwt-secret-32chars!!"

func hashPw(pw string) string {
	h, _ := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.MinCost)
	return string(h)
}

func makeUser(id, username, password string) *models.User {
	return &models.User{
		ID:           id,
		Username:     username,
		PasswordHash: hashPw(password),
		IsActive:     true,
		Roles:        []string{"operator"},
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

type testEnv struct {
	handler *handlers.AuthHandler
	authSvc *service.AuthService
}

func newTestEnv(users ...*models.User) *testEnv {
	userStore := newMemUserStore(users...)
	tokenStore := newMemTokenStore()
	authSvc := service.NewAuthService(context.Background(), userStore, tokenStore, handlerSecret)
	// UserService needs a RoleStore; pass a no-op for handler tests.
	userSvc := service.NewUserService(userStore, &noopRoleStore{})
	h := handlers.NewAuthHandler(authSvc, userSvc, slog.Default())
	return &testEnv{handler: h, authSvc: authSvc}
}

// noopRoleStore satisfies repository.RoleStore with stubs.
type noopRoleStore struct{}

func (n *noopRoleStore) Create(_ context.Context, _ *models.Role) error { return nil }
func (n *noopRoleStore) FindByID(_ context.Context, id string) (*models.Role, error) {
	return nil, repository.ErrNotFound
}
func (n *noopRoleStore) FindByName(_ context.Context, name string) (*models.Role, error) {
	return nil, repository.ErrNotFound
}
func (n *noopRoleStore) List(_ context.Context) ([]models.Role, error) { return nil, nil }
func (n *noopRoleStore) Update(_ context.Context, id string, _ map[string]interface{}) error {
	return nil
}
func (n *noopRoleStore) Delete(_ context.Context, id string) error { return nil }
func (n *noopRoleStore) Count(_ context.Context) (int64, error)    { return 0, nil }

func post(handler http.HandlerFunc, body any) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler(w, req)
	return w
}

func decodeBody(t *testing.T, rec *httptest.ResponseRecorder, dst any) {
	t.Helper()
	if err := json.NewDecoder(rec.Body).Decode(dst); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
}

// ── Login handler tests ───────────────────────────────────────────────────────

func TestLoginHandler_Success(t *testing.T) {
	env := newTestEnv(makeUser("u1", "alice", "pass"))
	rec := post(env.handler.Login, map[string]string{"username": "alice", "password": "pass"})

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d — body: %s", rec.Code, rec.Body.String())
	}

	var resp models.LoginResponse
	decodeBody(t, rec, &resp)
	if resp.AccessToken == "" {
		t.Error("access_token should be non-empty")
	}
	if resp.RefreshToken == "" {
		t.Error("refresh_token should be non-empty")
	}
}

func TestLoginHandler_WrongPassword(t *testing.T) {
	env := newTestEnv(makeUser("u1", "alice", "right"))
	rec := post(env.handler.Login, map[string]string{"username": "alice", "password": "wrong"})

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", rec.Code)
	}
}

func TestLoginHandler_MissingFields(t *testing.T) {
	env := newTestEnv()
	rec := post(env.handler.Login, map[string]string{"username": ""})

	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestLoginHandler_DisabledUser(t *testing.T) {
	u := makeUser("u1", "bob", "pass")
	u.IsActive = false
	env := newTestEnv(u)
	rec := post(env.handler.Login, map[string]string{"username": "bob", "password": "pass"})

	if rec.Code != http.StatusForbidden {
		t.Errorf("want 403, got %d", rec.Code)
	}
}

func TestLoginHandler_InvalidJSON(t *testing.T) {
	env := newTestEnv()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{bad json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.handler.Login(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

// ── Refresh handler tests ─────────────────────────────────────────────────────

func doLogin(t *testing.T, env *testEnv, username, password string) models.LoginResponse {
	t.Helper()
	rec := post(env.handler.Login, map[string]string{"username": username, "password": password})
	if rec.Code != http.StatusOK {
		t.Fatalf("login failed: %s", rec.Body.String())
	}
	var resp models.LoginResponse
	decodeBody(t, rec, &resp)
	return resp
}

func TestRefreshHandler_Success(t *testing.T) {
	env := newTestEnv(makeUser("u1", "alice", "pass"))
	login := doLogin(t, env, "alice", "pass")

	rec := post(env.handler.Refresh, map[string]string{"refresh_token": login.RefreshToken})
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d — body: %s", rec.Code, rec.Body.String())
	}

	var resp models.RefreshResponse
	decodeBody(t, rec, &resp)
	if resp.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
}

func TestRefreshHandler_MissingToken(t *testing.T) {
	env := newTestEnv()
	rec := post(env.handler.Refresh, map[string]string{"refresh_token": ""})

	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestRefreshHandler_InvalidToken(t *testing.T) {
	env := newTestEnv()
	rec := post(env.handler.Refresh, map[string]string{"refresh_token": "bogus-token"})

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", rec.Code)
	}
}

// ── Logout handler tests ──────────────────────────────────────────────────────

func TestLogoutHandler_Success(t *testing.T) {
	env := newTestEnv(makeUser("u1", "alice", "pass"))
	login := doLogin(t, env, "alice", "pass")

	rec := post(env.handler.Logout, map[string]string{"refresh_token": login.RefreshToken})
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}

	// Subsequent refresh must fail.
	rec2 := post(env.handler.Refresh, map[string]string{"refresh_token": login.RefreshToken})
	if rec2.Code != http.StatusUnauthorized {
		t.Errorf("refreshing after logout should return 401, got %d", rec2.Code)
	}
}

func TestLogoutHandler_InvalidJSON(t *testing.T) {
	env := newTestEnv()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.handler.Logout(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

// ── Me handler tests ──────────────────────────────────────────────────────────

func TestMeHandler_WithClaims(t *testing.T) {
	u := makeUser("u1", "alice", "pass")
	env := newTestEnv(u)
	login := doLogin(t, env, "alice", "pass")

	// Use the real RequireAuth middleware so claims are injected via the private context key.
	authn := middleware.NewAuthenticator(env.authSvc)
	protected := authn.RequireAuth(http.HandlerFunc(env.handler.Me))

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer "+login.AccessToken)
	w := httptest.NewRecorder()
	protected.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestMeHandler_NoClaims(t *testing.T) {
	env := newTestEnv()
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	w := httptest.NewRecorder()
	env.handler.Me(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", w.Code)
	}
}

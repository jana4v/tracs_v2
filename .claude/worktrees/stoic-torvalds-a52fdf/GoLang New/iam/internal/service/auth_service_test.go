package service_test

import (
	"context"
	"errors"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/mainframe/tm-system/iam/internal/models"
	"github.com/mainframe/tm-system/iam/internal/repository"
	"github.com/mainframe/tm-system/iam/internal/service"
)

// ── in-memory mock stores ─────────────────────────────────────────────────────

type mockUserStore struct {
	users map[string]*models.User // keyed by username
	byID  map[string]*models.User // keyed by ID
}

func newMockUserStore(users ...*models.User) *mockUserStore {
	m := &mockUserStore{
		users: make(map[string]*models.User),
		byID:  make(map[string]*models.User),
	}
	for _, u := range users {
		m.users[u.Username] = u
		m.byID[u.ID] = u
	}
	return m
}

func (m *mockUserStore) FindByUsername(_ context.Context, username string) (*models.User, error) {
	u, ok := m.users[username]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return u, nil
}

func (m *mockUserStore) FindByID(_ context.Context, id string) (*models.User, error) {
	u, ok := m.byID[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return u, nil
}

func (m *mockUserStore) Create(_ context.Context, u *models.User) error {
	if _, exists := m.users[u.Username]; exists {
		return repository.ErrDuplicateKey
	}
	m.users[u.Username] = u
	m.byID[u.ID] = u
	return nil
}

func (m *mockUserStore) List(_ context.Context) ([]models.User, error) { return nil, nil }
func (m *mockUserStore) Update(_ context.Context, id string, fields map[string]interface{}) error {
	u, ok := m.byID[id]
	if !ok {
		return repository.ErrNotFound
	}
	if pw, ok := fields["password_hash"]; ok {
		u.PasswordHash = pw.(string)
	}
	return nil
}
func (m *mockUserStore) Delete(_ context.Context, id string) error          { return nil }
func (m *mockUserStore) Count(_ context.Context) (int64, error)             { return 0, nil }
func (m *mockUserStore) UpdateLastLogin(_ context.Context, id string) error { return nil }

// ─────────────────────────────────────────────────────────────────────────────

type mockTokenStore struct {
	tokens map[string]*models.RefreshToken
}

func newMockTokenStore() *mockTokenStore {
	return &mockTokenStore{tokens: make(map[string]*models.RefreshToken)}
}

func (m *mockTokenStore) Create(_ context.Context, t *models.RefreshToken) error {
	m.tokens[t.Token] = t
	return nil
}

func (m *mockTokenStore) FindByToken(_ context.Context, token string) (*models.RefreshToken, error) {
	t, ok := m.tokens[token]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return t, nil
}

func (m *mockTokenStore) Revoke(_ context.Context, token string) error {
	t, ok := m.tokens[token]
	if !ok {
		return repository.ErrNotFound
	}
	t.Revoked = true
	return nil
}

func (m *mockTokenStore) RevokeAllForUser(_ context.Context, userID string) error {
	for _, t := range m.tokens {
		if t.UserID == userID {
			t.Revoked = true
		}
	}
	return nil
}

func (m *mockTokenStore) DeleteExpired(_ context.Context) error { return nil }

// ── helpers ───────────────────────────────────────────────────────────────────

func mustHash(password string) string {
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		panic(err)
	}
	return string(h)
}

func activeUser(id, username, password string) *models.User {
	return &models.User{
		ID:           id,
		Username:     username,
		PasswordHash: mustHash(password),
		IsActive:     true,
		Roles:        []string{"operator"},
	}
}

const testSecret = "test-jwt-secret-at-least-32-chars"

func newService(users *mockUserStore, tokens *mockTokenStore) *service.AuthService {
	return service.NewAuthService(context.Background(), users, tokens, testSecret)
}

// ── Login tests ───────────────────────────────────────────────────────────────

func TestLogin_Success(t *testing.T) {
	u := activeUser("u1", "alice", "s3cr3t")
	svc := newService(newMockUserStore(u), newMockTokenStore())

	resp, err := svc.Login(context.Background(), "alice", "s3cr3t")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
	if resp.RefreshToken == "" {
		t.Error("expected non-empty refresh token")
	}
	if resp.TokenType != "Bearer" {
		t.Errorf("expected Bearer, got %q", resp.TokenType)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	u := activeUser("u1", "alice", "correct")
	svc := newService(newMockUserStore(u), newMockTokenStore())

	_, err := svc.Login(context.Background(), "alice", "wrong")
	if !errors.Is(err, service.ErrInvalidCredentials) {
		t.Errorf("want ErrInvalidCredentials, got %v", err)
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	svc := newService(newMockUserStore(), newMockTokenStore())

	_, err := svc.Login(context.Background(), "nobody", "x")
	if !errors.Is(err, service.ErrInvalidCredentials) {
		t.Errorf("want ErrInvalidCredentials, got %v", err)
	}
}

func TestLogin_DisabledAccount(t *testing.T) {
	u := activeUser("u1", "bob", "pass")
	u.IsActive = false
	svc := newService(newMockUserStore(u), newMockTokenStore())

	_, err := svc.Login(context.Background(), "bob", "pass")
	if !errors.Is(err, service.ErrAccountDisabled) {
		t.Errorf("want ErrAccountDisabled, got %v", err)
	}
}

// ── Refresh tests ─────────────────────────────────────────────────────────────

func TestRefresh_Success(t *testing.T) {
	u := activeUser("u1", "alice", "s3cr3t")
	tokens := newMockTokenStore()
	svc := newService(newMockUserStore(u), tokens)

	loginResp, _ := svc.Login(context.Background(), "alice", "s3cr3t")

	resp, err := svc.Refresh(context.Background(), loginResp.RefreshToken)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("expected non-empty access token in refresh response")
	}
}

func TestRefresh_RevokedToken(t *testing.T) {
	u := activeUser("u1", "alice", "s3cr3t")
	tokens := newMockTokenStore()
	svc := newService(newMockUserStore(u), tokens)

	loginResp, _ := svc.Login(context.Background(), "alice", "s3cr3t")
	_ = svc.Logout(context.Background(), loginResp.RefreshToken)

	_, err := svc.Refresh(context.Background(), loginResp.RefreshToken)
	if !errors.Is(err, service.ErrTokenExpired) {
		t.Errorf("want ErrTokenExpired for revoked token, got %v", err)
	}
}

func TestRefresh_UnknownToken(t *testing.T) {
	svc := newService(newMockUserStore(), newMockTokenStore())

	_, err := svc.Refresh(context.Background(), "does-not-exist")
	if !errors.Is(err, service.ErrTokenRevoked) {
		t.Errorf("want ErrTokenRevoked, got %v", err)
	}
}

// ── Logout tests ──────────────────────────────────────────────────────────────

func TestLogout_Success(t *testing.T) {
	u := activeUser("u1", "alice", "s3cr3t")
	tokens := newMockTokenStore()
	svc := newService(newMockUserStore(u), tokens)

	loginResp, _ := svc.Login(context.Background(), "alice", "s3cr3t")

	if err := svc.Logout(context.Background(), loginResp.RefreshToken); err != nil {
		t.Fatalf("expected no error on logout, got %v", err)
	}
}

func TestLogout_Idempotent(t *testing.T) {
	svc := newService(newMockUserStore(), newMockTokenStore())

	// Logging out a non-existent token should not error (idempotent).
	if err := svc.Logout(context.Background(), "ghost-token"); err != nil {
		t.Errorf("expected nil (idempotent), got %v", err)
	}
}

// ── ValidateAccessToken tests ─────────────────────────────────────────────────

func TestValidateAccessToken_Valid(t *testing.T) {
	u := activeUser("u1", "alice", "s3cr3t")
	tokens := newMockTokenStore()
	svc := newService(newMockUserStore(u), tokens)

	loginResp, _ := svc.Login(context.Background(), "alice", "s3cr3t")

	claims, err := svc.ValidateAccessToken(loginResp.AccessToken)
	if err != nil {
		t.Fatalf("expected valid token, got %v", err)
	}
	if claims.Username != "alice" {
		t.Errorf("expected username alice, got %q", claims.Username)
	}
}

func TestValidateAccessToken_Tampered(t *testing.T) {
	svc := newService(newMockUserStore(), newMockTokenStore())

	_, err := svc.ValidateAccessToken("not.a.valid.jwt")
	if !errors.Is(err, service.ErrTokenInvalid) {
		t.Errorf("want ErrTokenInvalid, got %v", err)
	}
}

func TestValidateAccessToken_WrongSecret(t *testing.T) {
	u := activeUser("u1", "alice", "s3cr3t")
	svcA := newService(newMockUserStore(u), newMockTokenStore())
	svcB := service.NewAuthService(context.Background(), newMockUserStore(u), newMockTokenStore(), "completely-different-secret-xyz")

	loginResp, _ := svcA.Login(context.Background(), "alice", "s3cr3t")

	_, err := svcB.ValidateAccessToken(loginResp.AccessToken)
	if !errors.Is(err, service.ErrTokenInvalid) {
		t.Errorf("want ErrTokenInvalid for wrong-secret token, got %v", err)
	}
}

// ── ChangePassword tests ──────────────────────────────────────────────────────

func TestChangePassword_Success(t *testing.T) {
	u := activeUser("u1", "alice", "old-pass")
	tokens := newMockTokenStore()
	svc := newService(newMockUserStore(u), tokens)

	if err := svc.ChangePassword(context.Background(), "u1", "old-pass", "new-pass"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Old password must now fail.
	_, err := svc.Login(context.Background(), "alice", "old-pass")
	if !errors.Is(err, service.ErrInvalidCredentials) {
		t.Errorf("old password should no longer work, got %v", err)
	}

	// New password must succeed.
	_, err = svc.Login(context.Background(), "alice", "new-pass")
	if err != nil {
		t.Errorf("new password should work, got %v", err)
	}
}

func TestChangePassword_WrongOldPassword(t *testing.T) {
	u := activeUser("u1", "alice", "real-pass")
	svc := newService(newMockUserStore(u), newMockTokenStore())

	err := svc.ChangePassword(context.Background(), "u1", "wrong-old", "new-pass")
	if !errors.Is(err, service.ErrInvalidCredentials) {
		t.Errorf("want ErrInvalidCredentials, got %v", err)
	}
}

func TestChangePassword_UserNotFound(t *testing.T) {
	svc := newService(newMockUserStore(), newMockTokenStore())

	err := svc.ChangePassword(context.Background(), "ghost-id", "any", "any")
	if err == nil {
		t.Error("expected error for unknown user ID")
	}
}

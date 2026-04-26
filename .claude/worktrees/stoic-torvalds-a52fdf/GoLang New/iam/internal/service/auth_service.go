package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/mainframe/tm-system/iam/internal/models"
	"github.com/mainframe/tm-system/iam/internal/repository"
)

// Sentinel errors for authentication failures.
var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrAccountDisabled    = errors.New("account is disabled")
	ErrTokenExpired       = errors.New("token has expired")
	ErrTokenInvalid       = errors.New("token is invalid")
	ErrTokenRevoked       = errors.New("token has been revoked")
)

const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 7 * 24 * time.Hour
	bcryptCost      = 12
)

// Claims are the JWT payload fields for access tokens.
type Claims struct {
	UserID   string   `json:"uid"`
	Username string   `json:"sub"`
	Roles    []string `json:"roles"`
	jwt.RegisteredClaims
}

// AuthService handles login, token issuance, and token validation.
type AuthService struct {
	users  repository.UserStore
	tokens repository.TokenStore
	secret []byte
	svcCtx context.Context // used for fire-and-forget background calls
}

// NewAuthService creates an AuthService with the given JWT signing secret.
// svcCtx should be the application-lifetime context (e.g. from main); it is
// used for background goroutines so they are cancelled on shutdown instead of
// continuing with context.Background().
func NewAuthService(
	svcCtx context.Context,
	users repository.UserStore,
	tokens repository.TokenStore,
	jwtSecret string,
) *AuthService {
	return &AuthService{
		users:  users,
		tokens: tokens,
		secret: []byte(jwtSecret),
		svcCtx: svcCtx,
	}
}

// Login validates credentials and returns a LoginResponse containing both tokens.
func (s *AuthService) Login(ctx context.Context, username, password string) (*models.LoginResponse, error) {
	user, err := s.users.FindByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("auth login lookup: %w", err)
	}

	if !user.IsActive {
		return nil, ErrAccountDisabled
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	accessToken, err := s.issueAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("issue access token: %w", err)
	}

	refreshToken, err := s.issueRefreshToken(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("issue refresh token: %w", err)
	}

	// Update last_login asynchronously so it does not block the login response.
	// svcCtx is the application-lifetime context, not the request context, so
	// this goroutine is cancelled cleanly on shutdown rather than running forever.
	go func() { _ = s.users.UpdateLastLogin(s.svcCtx, user.ID) }()

	return &models.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(accessTokenTTL.Seconds()),
		User:         user.ToResponse(),
	}, nil
}

// Refresh validates a refresh token and issues a new access token.
func (s *AuthService) Refresh(ctx context.Context, rawRefreshToken string) (*models.RefreshResponse, error) {
	stored, err := s.tokens.FindByToken(ctx, rawRefreshToken)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrTokenRevoked
		}
		return nil, fmt.Errorf("refresh token lookup: %w", err)
	}

	if stored.Revoked || time.Now().After(stored.ExpiresAt) {
		return nil, ErrTokenExpired
	}

	user, err := s.users.FindByID(ctx, stored.UserID)
	if err != nil {
		return nil, fmt.Errorf("refresh user lookup: %w", err)
	}
	if !user.IsActive {
		return nil, ErrAccountDisabled
	}

	accessToken, err := s.issueAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("issue access token on refresh: %w", err)
	}

	return &models.RefreshResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int(accessTokenTTL.Seconds()),
	}, nil
}

// Logout revokes the given refresh token.
func (s *AuthService) Logout(ctx context.Context, rawRefreshToken string) error {
	if err := s.tokens.Revoke(ctx, rawRefreshToken); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil // already gone – idempotent
		}
		return fmt.Errorf("logout revoke: %w", err)
	}
	return nil
}

// ValidateAccessToken parses and validates a JWT access token, returning its claims.
func (s *AuthService) ValidateAccessToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrTokenInvalid
	}
	return claims, nil
}

// ChangePassword validates the old password and sets a new bcrypt hash.
func (s *AuthService) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("change password lookup: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword)); err != nil {
		return ErrInvalidCredentials
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcryptCost)
	if err != nil {
		return fmt.Errorf("hash new password: %w", err)
	}

	// Revoke all existing refresh tokens to force re-login everywhere.
	_ = s.tokens.RevokeAllForUser(ctx, user.ID)

	return s.users.Update(ctx, userID, map[string]interface{}{
		"password_hash": string(hash),
	})
}

// HashPassword is a utility used when creating users.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}

// --- private helpers ---------------------------------------------------------

func (s *AuthService) issueAccessToken(user *models.User) (string, error) {
	now := time.Now()
	claims := &Claims{
		UserID:   user.ID,
		Username: user.Username,
		Roles:    user.Roles,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenTTL)),
			Issuer:    "mainframe-iam",
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
}

func (s *AuthService) issueRefreshToken(ctx context.Context, user *models.User) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate refresh token bytes: %w", err)
	}
	tokenStr := hex.EncodeToString(raw)

	rt := &models.RefreshToken{
		UserID:    user.ID,
		Token:     tokenStr,
		ExpiresAt: time.Now().Add(refreshTokenTTL),
	}
	if err := s.tokens.Create(ctx, rt); err != nil {
		return "", err
	}
	return tokenStr, nil
}

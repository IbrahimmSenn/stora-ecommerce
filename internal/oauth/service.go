// service.go — OAuth login logic: find existing user, link account, or create new user.
package oauth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/user"
)

// TokenPair mirrors auth.LoginResponse for the OAuth flow.
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"`
}

// TokenGenerator creates JWT token pairs (injected to avoid import cycles with auth).
type TokenGenerator func(userID, email, role, secret string) (accessToken, refreshToken string, err error)

// RefreshTokenStorer stores refresh tokens in the database.
type RefreshTokenStorer func(ctx context.Context, token string, userID uuid.UUID) error

type Service interface {
	OAuthLogin(ctx context.Context, info *UserInfo) (*TokenPair, error)
}

type service struct {
	userRepo     user.UserRepository
	oauthRepo    Repository
	generateJWT  TokenGenerator
	storeRefresh RefreshTokenStorer
	jwtSecret    string
}

func NewService(
	userRepo user.UserRepository,
	oauthRepo Repository,
	generateJWT TokenGenerator,
	storeRefresh RefreshTokenStorer,
	jwtSecret string,
) Service {
	return &service{
		userRepo:     userRepo,
		oauthRepo:    oauthRepo,
		generateJWT:  generateJWT,
		storeRefresh: storeRefresh,
		jwtSecret:    jwtSecret,
	}
}

// OAuthLogin handles the "find or create user" flow for OAuth.
// 1. Check if this OAuth account already exists → log in the linked user.
// 2. Check if a user with this email exists → link the OAuth account.
// 3. Otherwise → create a new user + link the OAuth account.
func (s *service) OAuthLogin(ctx context.Context, info *UserInfo) (*TokenPair, error) {
	email := strings.ToLower(strings.TrimSpace(info.Email))

	// Check if OAuth account already linked.
	existingUserID, err := s.oauthRepo.GetUserIDByProvider(ctx, info.Provider, info.ProviderUserID)
	if err == nil && existingUserID != "" {
		return s.issueTokens(ctx, existingUserID)
	}

	// Check if user with this email exists.
	u, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil && !errors.Is(err, user.ErrUserNotFound) {
		return nil, fmt.Errorf("oauth lookup: %w", err)
	}

	if u != nil {
		// Link the OAuth account to the existing user.
		if err := s.oauthRepo.LinkAccount(ctx, u.Id.String(), info.Provider, info.ProviderUserID); err != nil {
			return nil, fmt.Errorf("oauth link: %w", err)
		}
		return s.issueTokens(ctx, u.Id.String())
	}

	// Create a new user (no password — OAuth-only account).
	newUser := user.User{
		Id:    uuid.New(),
		Email: email,
		Role:  "customer",
	}
	if err := s.userRepo.CreateOAuthUser(ctx, newUser); err != nil {
		return nil, fmt.Errorf("oauth create user: %w", err)
	}

	if err := s.oauthRepo.LinkAccount(ctx, newUser.Id.String(), info.Provider, info.ProviderUserID); err != nil {
		return nil, fmt.Errorf("oauth link new: %w", err)
	}

	return s.issueTokens(ctx, newUser.Id.String())
}

func (s *service) issueTokens(ctx context.Context, userID string) (*TokenPair, error) {
	u, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	access, refresh, err := s.generateJWT(u.Id.String(), u.Email, u.Role, s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("generate tokens: %w", err)
	}

	if err := s.storeRefresh(ctx, refresh, u.Id); err != nil {
		return nil, fmt.Errorf("store refresh: %w", err)
	}

	return &TokenPair{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresAt:    time.Now().Add(15 * time.Minute),
		TokenType:    "Bearer",
	}, nil
}

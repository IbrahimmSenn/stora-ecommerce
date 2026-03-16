package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/user"
)

type AuthService interface {
	Login(ctx context.Context, req LoginRequest) (*LoginResponse, error)
	RefreshTokens(ctx context.Context, req RefreshRequest) (*LoginResponse, error)
	Logout(ctx context.Context, userID string) error
}

type authService struct {
	userRepo  user.UserRepository
	authRepo  AuthRepository
	jwtSecret string
	validate  *validator.Validate
}

func NewService(userRepo user.UserRepository, authRepo AuthRepository, jwtSecret string) AuthService {
	return &authService{
		userRepo:  userRepo,
		authRepo:  authRepo,
		jwtSecret: jwtSecret,
		validate:  validator.New(),
	}
}

func (s *authService) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	if err := s.validate.Struct(req); err != nil {
		return nil, err
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))

	u, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("login: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	tokenPair, err := GenerateTokenPair(u.Id.String(), u.Email, "user", s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("generate tokens: %w", err)
	}

	refreshToken := RefreshToken{
		ID:        uuid.New(),
		Token:     tokenPair.RefreshToken,
		UserID:    u.Id,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	if err := s.authRepo.StoreRefreshToken(ctx, refreshToken); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}

	return &LoginResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    time.Now().Add(15 * time.Minute),
		TokenType:    "Bearer",
	}, nil
}

// RefreshTokens implements single-use refresh token rotation.
// The old token is marked as used, and a brand new token pair is issued.
// If a used token is presented, all of the user's tokens are revoked (replay detection).
func (s *authService) RefreshTokens(ctx context.Context, req RefreshRequest) (*LoginResponse, error) {
	if err := s.validate.Struct(req); err != nil {
		return nil, err
	}

	stored, err := s.authRepo.GetRefreshToken(ctx, req.RefreshToken)
	if err != nil {
		if errors.Is(err, ErrTokenNotFound) {
			return nil, ErrInvalidToken
		}
		return nil, fmt.Errorf("refresh tokens: %w", err)
	}

	// Replay detection: if someone reuses an already-used token,
	// revoke the entire token family for this user.
	if stored.Used {
		_ = s.authRepo.RevokeAllUserTokens(ctx, stored.UserID.String())
		return nil, ErrTokenUsed
	}

	if stored.Revoked {
		return nil, ErrTokenRevoked
	}

	if time.Now().After(stored.ExpiresAt) {
		return nil, ErrExpiredToken
	}

	// Mark the current refresh token as used (single-use enforcement).
	if err := s.authRepo.MarkRefreshTokenUsed(ctx, stored.ID.String()); err != nil {
		return nil, fmt.Errorf("mark token used: %w", err)
	}

	// Validate the JWT signature to extract the user's subject claim.
	claims, err := ValidateRefreshToken(req.RefreshToken, s.jwtSecret)
	if err != nil {
		return nil, err
	}

	userID := claims.Subject

	u, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("refresh tokens: %w", err)
	}

	// Issue a completely new token pair (rotation).
	tokenPair, err := GenerateTokenPair(u.Id.String(), u.Email, "user", s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("generate tokens: %w", err)
	}

	newRefreshToken := RefreshToken{
		ID:        uuid.New(),
		Token:     tokenPair.RefreshToken,
		UserID:    u.Id,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	if err := s.authRepo.StoreRefreshToken(ctx, newRefreshToken); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}

	return &LoginResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    time.Now().Add(15 * time.Minute),
		TokenType:    "Bearer",
	}, nil
}

// Logout revokes all refresh tokens for the user, effectively signing them out
// of all sessions. The caller must be authenticated (userID comes from the JWT).
func (s *authService) Logout(ctx context.Context, userID string) error {
	if err := s.authRepo.RevokeAllUserTokens(ctx, userID); err != nil {
		return fmt.Errorf("logout: %w", err)
	}
	return nil
}

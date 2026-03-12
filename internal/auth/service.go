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

// service.go — registration logic: input validation, captcha check, password hashing.
package user

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/captcha"
)

type UserService interface {
	Register(ctx context.Context, req RegisterRequest) (*UserResponse, error)
}

type userService struct {
	repo       UserRepository
	bcryptCost int
	validate   *validator.Validate
	captcha    *captcha.Verifier
}

func NewService(repo UserRepository, cost int, captchaVerifier *captcha.Verifier) UserService {
	if cost == 0 {
		cost = bcrypt.DefaultCost
	}
	return &userService{
		repo:       repo,
		bcryptCost: cost,
		validate:   validator.New(),
		captcha:    captchaVerifier,
	}
}

func (s *userService) Register(ctx context.Context, req RegisterRequest) (*UserResponse, error) {
	if err := s.validate.Struct(req); err != nil {
		return nil, err
	}

	// Verify captcha if configured.
	if s.captcha != nil {
		if err := s.captcha.Verify(req.CaptchaToken); err != nil {
			return nil, fmt.Errorf("captcha verification: %w", err)
		}
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), s.bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("secure password generation: %w", err)
	}

	user := User{
		Id:           uuid.New(),
		Email:        email,
		PasswordHash: string(hashedPassword),
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("user registration: %w", err)
	}

	return &UserResponse{
		Id:    user.Id,
		Email: user.Email,
	}, nil
}

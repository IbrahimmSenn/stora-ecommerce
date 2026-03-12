package user

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repo       UserRepository
	bcryptCost int
	validate   *validator.Validate
}

func NewService(repo UserRepository, cost int) *Service {
	if cost == 0 {
		cost = bcrypt.DefaultCost
	}
	return &Service{
		repo:       repo,
		bcryptCost: cost,
		validate:   validator.New(),
	}
}

func (s *Service) Register(ctx context.Context, req RegisterRequest) (*UserResponse, error) {

	if err := s.validate.Struct(req); err != nil {
		return nil, err
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

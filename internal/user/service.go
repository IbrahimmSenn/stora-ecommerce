// service.go — registration: validates input, checks captcha, hashes password.
package user

import (
	"context"
	"fmt"
	"log"
	"strings"
	"unicode/utf8"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/IbrahimmSenn/stora-ecommerce/internal/captcha"
	"github.com/IbrahimmSenn/stora-ecommerce/internal/passwordpolicy"
)

type UserService interface {
	Register(ctx context.Context, req RegisterRequest) (*UserResponse, error)
	GetMe(ctx context.Context, userID string) (*Me, error)
	UpdateProfile(ctx context.Context, userID string, req UpdateProfileRequest) (*Me, error)
	ChangePassword(ctx context.Context, userID string, req ChangePasswordRequest) error
	AdminListUsers(ctx context.Context, page, pageSize int) (*AdminUserList, error)
	AdminSetRole(ctx context.Context, userID, role string) error
}

// TokenRevoker invalidates a user's refresh tokens after a password change.
// Narrow interface so the user service stays decoupled from the auth package;
// satisfied by auth's repository.
type TokenRevoker interface {
	RevokeAllUserTokens(ctx context.Context, userID string) error
}

type userService struct {
	repo       UserRepository
	bcryptCost int
	validate   *validator.Validate
	captcha    *captcha.Verifier
	revoker    TokenRevoker
}

func NewService(repo UserRepository, cost int, captchaVerifier *captcha.Verifier, revoker TokenRevoker) UserService {
	if cost == 0 {
		cost = bcrypt.DefaultCost
	}
	return &userService{
		repo:       repo,
		bcryptCost: cost,
		validate:   validator.New(),
		captcha:    captchaVerifier,
		revoker:    revoker,
	}
}

func (s *userService) Register(ctx context.Context, req RegisterRequest) (*UserResponse, error) {
	if err := s.validate.Struct(req); err != nil {
		return nil, err
	}
	if err := passwordpolicy.Validate(req.Password); err != nil {
		return nil, err
	}

	// Verify captcha if configured. Wrap with ErrCaptchaInvalid so the
	// handler can map it to 400, and preserve the underlying reason for logs.
	if s.captcha != nil {
		if err := s.captcha.Verify(req.CaptchaToken); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrCaptchaInvalid, err)
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

func toMe(u *User) *Me {
	return &Me{Id: u.Id, Email: u.Email, Name: u.Name, Role: u.Role, CreatedAt: u.CreatedAt}
}

func (s *userService) GetMe(ctx context.Context, userID string) (*Me, error) {
	u, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return toMe(u), nil
}

func (s *userService) UpdateProfile(ctx context.Context, userID string, req UpdateProfileRequest) (*Me, error) {
	name := strings.TrimSpace(req.Name)
	if utf8.RuneCountInString(name) > 100 {
		return nil, ErrNameTooLong
	}
	if err := s.repo.UpdateName(ctx, userID, name); err != nil {
		return nil, err
	}
	return s.GetMe(ctx, userID)
}

func (s *userService) ChangePassword(ctx context.Context, userID string, req ChangePasswordRequest) error {
	if err := s.validate.Struct(req); err != nil {
		return err
	}
	u, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if u.PasswordHash == "" {
		return ErrNoPassword
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.CurrentPassword)) != nil {
		return ErrWrongPassword
	}
	if err := passwordpolicy.Validate(req.NewPassword); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), s.bcryptCost)
	if err != nil {
		return fmt.Errorf("secure password generation: %w", err)
	}
	if err := s.repo.UpdatePassword(ctx, userID, string(hash)); err != nil {
		return err
	}
	// Same semantics as the reset flow: other sessions lose their refresh
	// tokens; best-effort so a revocation hiccup doesn't fail the change.
	if s.revoker != nil {
		if err := s.revoker.RevokeAllUserTokens(ctx, userID); err != nil {
			log.Printf("change password: revoke tokens for %s: %v", userID, err)
		}
	}
	return nil
}

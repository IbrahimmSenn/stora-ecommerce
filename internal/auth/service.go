// service.go — login, token refresh with replay detection, password reset, and 2FA.
package auth

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base32"
	"encoding/base64"
	"errors"
	"fmt"
	"image/png"
	"log/slog"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"

	"github.com/IbrahimmSenn/stora-ecommerce/internal/mailer"
	"github.com/IbrahimmSenn/stora-ecommerce/internal/metrics"
	"github.com/IbrahimmSenn/stora-ecommerce/internal/passwordpolicy"
	"github.com/IbrahimmSenn/stora-ecommerce/internal/user"
)

type AuthService interface {
	Login(ctx context.Context, req LoginRequest) (*LoginResponse, error)
	RefreshTokens(ctx context.Context, req RefreshRequest) (*LoginResponse, error)
	Logout(ctx context.Context, userID string) error

	// Password reset
	ForgotPassword(ctx context.Context, req ForgotPasswordRequest) error
	ResetPassword(ctx context.Context, req ResetPasswordRequest) error

	// 2FA
	Setup2FA(ctx context.Context, userID, email string) (*Setup2FAResponse, error)
	Enable2FA(ctx context.Context, userID string, req Verify2FARequest) error
	Disable2FA(ctx context.Context, userID string, req Verify2FARequest) error
}

type authService struct {
	userRepo   user.UserRepository
	authRepo   AuthRepository
	jwtSecret  string
	validate   *validator.Validate
	mailer     *mailer.Mailer
	baseURL    string
	bcryptCost int
	metrics    metrics.Recorder
}

func NewService(userRepo user.UserRepository, authRepo AuthRepository, jwtSecret string, opts ...ServiceOption) AuthService {
	s := &authService{
		userRepo:   userRepo,
		authRepo:   authRepo,
		jwtSecret:  jwtSecret,
		validate:   validator.New(),
		bcryptCost: bcrypt.DefaultCost,
		metrics:    metrics.Noop{},
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

type ServiceOption func(*authService)

func WithMailer(m *mailer.Mailer) ServiceOption {
	return func(s *authService) { s.mailer = m }
}

func WithBaseURL(url string) ServiceOption {
	return func(s *authService) { s.baseURL = url }
}

func WithBcryptCost(cost int) ServiceOption {
	return func(s *authService) { s.bcryptCost = cost }
}

func WithMetrics(r metrics.Recorder) ServiceOption {
	return func(s *authService) { s.metrics = r }
}

// --- Login ---

func (s *authService) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	if err := s.validate.Struct(req); err != nil {
		return nil, err
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))

	u, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			s.metrics.LoginAttempt("failure", "invalid_credentials")
			slog.Warn("login_failed", "reason", "invalid_credentials")
			return nil, ErrInvalidCredentials
		}
		s.metrics.LoginAttempt("failure", "error")
		return nil, fmt.Errorf("login: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)); err != nil {
		s.metrics.LoginAttempt("failure", "invalid_credentials")
		slog.Warn("login_failed", "reason", "invalid_credentials")
		return nil, ErrInvalidCredentials
	}

	// Check if 2FA is enabled for this user.
	tfa, err := s.authRepo.Get2FAByUserID(ctx, u.Id.String())
	twoFactorEnabled := err == nil && tfa.IsEnabled
	if twoFactorEnabled {
		// 2FA is enabled — require TOTP code.
		if req.TOTPCode == "" {
			// Password was right; the client is being sent to the TOTP prompt.
			// Tracked as its own result so 2FA challenges never look like
			// brute-force failures on the security dashboard.
			s.metrics.LoginAttempt("challenge", "2fa_required")
			return nil, Err2FARequired
		}

		valid := totp.Validate(req.TOTPCode, tfa.SecretKey)
		if !valid {
			// Check recovery codes as fallback.
			if !s.useRecoveryCode(ctx, u.Id.String(), tfa, req.TOTPCode) {
				s.metrics.LoginAttempt("failure", "invalid_2fa")
				slog.Warn("login_failed", "reason", "invalid_2fa")
				return nil, ErrInvalid2FACode
			}
		}
	}

	// Staff accounts must run with 2FA. Signal the frontend to send them to
	// setup; admin endpoints remain blocked server-side until it's enabled.
	staffNeeds2FA := isStaffRole(u.Role) && !twoFactorEnabled

	tokenPair, err := GenerateTokenPair(u.Id.String(), u.Email, u.Role, s.jwtSecret)
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

	s.metrics.LoginAttempt("success", "")
	return &LoginResponse{
		AccessToken:            tokenPair.AccessToken,
		RefreshToken:           tokenPair.RefreshToken,
		ExpiresAt:              time.Now().Add(15 * time.Minute),
		TokenType:              "Bearer",
		TwoFactorSetupRequired: staffNeeds2FA,
	}, nil
}

// isStaffRole reports whether a role is one of the privileged staff roles that
// must have 2FA enabled.
func isStaffRole(role string) bool {
	return role == "admin" || role == "support" || role == "sales"
}

// --- Refresh tokens ---

func (s *authService) RefreshTokens(ctx context.Context, req RefreshRequest) (*LoginResponse, error) {
	if err := s.validate.Struct(req); err != nil {
		return nil, err
	}

	stored, err := s.authRepo.GetRefreshToken(ctx, req.RefreshToken)
	if err != nil {
		if errors.Is(err, ErrTokenNotFound) {
			s.metrics.TokenRefresh("failure")
			return nil, ErrInvalidToken
		}
		s.metrics.TokenRefresh("failure")
		return nil, fmt.Errorf("refresh tokens: %w", err)
	}

	if stored.Used {
		// Replay of a rotated token — possible theft, so the whole family is
		// revoked. Logged for the security dashboard.
		slog.Warn("refresh_token_reuse", "user_id", stored.UserID.String())
		s.metrics.TokenRefresh("failure")
		_ = s.authRepo.RevokeAllUserTokens(ctx, stored.UserID.String())
		return nil, ErrTokenUsed
	}

	if stored.Revoked {
		s.metrics.TokenRefresh("failure")
		return nil, ErrTokenRevoked
	}

	if time.Now().After(stored.ExpiresAt) {
		s.metrics.TokenRefresh("failure")
		return nil, ErrExpiredToken
	}

	if err := s.authRepo.MarkRefreshTokenUsed(ctx, stored.ID.String()); err != nil {
		// Lost the rotation race (or a replay slipped past the read-time check):
		// the token was already consumed. Treat exactly like reuse — revoke the
		// family so a stolen token can't be leveraged.
		if errors.Is(err, ErrTokenUsed) {
			slog.Warn("refresh_token_reuse", "user_id", stored.UserID.String(), "reason", "rotation_race")
			s.metrics.TokenRefresh("failure")
			_ = s.authRepo.RevokeAllUserTokens(ctx, stored.UserID.String())
			return nil, ErrTokenUsed
		}
		return nil, fmt.Errorf("mark token used: %w", err)
	}

	claims, err := ValidateRefreshToken(req.RefreshToken, s.jwtSecret)
	if err != nil {
		return nil, err
	}

	userID := claims.Subject

	u, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("refresh tokens: %w", err)
	}

	tokenPair, err := GenerateTokenPair(u.Id.String(), u.Email, u.Role, s.jwtSecret)
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

	s.metrics.TokenRefresh("success")
	return &LoginResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    time.Now().Add(15 * time.Minute),
		TokenType:    "Bearer",
	}, nil
}

// --- Logout ---

func (s *authService) Logout(ctx context.Context, userID string) error {
	if err := s.authRepo.RevokeAllUserTokens(ctx, userID); err != nil {
		return fmt.Errorf("logout: %w", err)
	}
	return nil
}

// --- Password reset ---

func (s *authService) ForgotPassword(ctx context.Context, req ForgotPasswordRequest) error {
	if err := s.validate.Struct(req); err != nil {
		return err
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))

	// Counted before the user lookup so enumeration probes show up too.
	s.metrics.PasswordReset("requested")
	slog.Info("password_reset_requested")

	// Always return success to prevent email enumeration.
	u, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil
	}

	// Generate a cryptographically secure token.
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return fmt.Errorf("generate reset token: %w", err)
	}
	tokenString := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(tokenBytes)

	resetToken := PasswordResetToken{
		ID:        uuid.New(),
		UserID:    u.Id,
		Token:     tokenString,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	if err := s.authRepo.StorePasswordResetToken(ctx, resetToken); err != nil {
		return fmt.Errorf("store reset token: %w", err)
	}

	resetLink := fmt.Sprintf("%s/reset-password?token=%s", s.baseURL, tokenString)
	body := fmt.Sprintf(`<h2>Password Reset</h2>
<p>You requested a password reset. Click the link below to set a new password:</p>
<p><a href="%s">Reset Password</a></p>
<p>This link expires in 1 hour. If you didn't request this, ignore this email.</p>`, resetLink)

	if s.mailer != nil {
		if err := s.mailer.Send(email, "Password Reset", body); err != nil {
			return fmt.Errorf("send reset email: %w", err)
		}
	}

	return nil
}

func (s *authService) ResetPassword(ctx context.Context, req ResetPasswordRequest) error {
	if err := s.validate.Struct(req); err != nil {
		return err
	}
	if err := passwordpolicy.Validate(req.NewPassword); err != nil {
		return err
	}

	stored, err := s.authRepo.GetPasswordResetToken(ctx, req.Token)
	if err != nil {
		s.metrics.PasswordReset("failed")
		return err
	}

	if stored.Used {
		s.metrics.PasswordReset("failed")
		return ErrResetTokenUsed
	}

	if time.Now().After(stored.ExpiresAt) {
		s.metrics.PasswordReset("failed")
		return ErrResetTokenExpired
	}

	// Hash the new password.
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), s.bcryptCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	// Update the user's password.
	if err := s.userRepo.UpdatePassword(ctx, stored.UserID.String(), string(hashedPassword)); err != nil {
		return fmt.Errorf("update password: %w", err)
	}

	// Mark token as used.
	if err := s.authRepo.MarkResetTokenUsed(ctx, stored.ID.String()); err != nil {
		return fmt.Errorf("mark reset token used: %w", err)
	}

	// Revoke all refresh tokens for security.
	_ = s.authRepo.RevokeAllUserTokens(ctx, stored.UserID.String())

	s.metrics.PasswordReset("completed")
	return nil
}

// --- 2FA ---

func (s *authService) Setup2FA(ctx context.Context, userID, email string) (*Setup2FAResponse, error) {
	// Check if 2FA is already set up.
	existing, err := s.authRepo.Get2FAByUserID(ctx, userID)
	if err == nil && existing.IsEnabled {
		return nil, Err2FAAlreadyEnabled
	}

	// If there's an existing non-enabled record, delete it so we can create fresh.
	if err == nil && !existing.IsEnabled {
		_ = s.authRepo.Delete2FA(ctx, userID)
	}

	// Generate TOTP key.
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "i-love-shopping",
		AccountName: email,
	})
	if err != nil {
		return nil, fmt.Errorf("generate totp key: %w", err)
	}

	// Generate QR code as base64 PNG.
	img, err := key.Image(200, 200)
	if err != nil {
		return nil, fmt.Errorf("generate qr code: %w", err)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("encode qr code: %w", err)
	}
	qrBase64 := base64.StdEncoding.EncodeToString(buf.Bytes())

	// Generate recovery codes.
	recoveryCodes, err := generateRecoveryCodes(8)
	if err != nil {
		return nil, fmt.Errorf("generate recovery codes: %w", err)
	}

	// Store in database (not enabled yet — user must verify first).
	tfa := TwoFactorAuth{
		ID:            uuid.New(),
		UserID:        uuid.MustParse(userID),
		SecretKey:     key.Secret(),
		RecoveryCodes: recoveryCodes,
	}

	if err := s.authRepo.Store2FA(ctx, tfa); err != nil {
		return nil, fmt.Errorf("store 2fa: %w", err)
	}

	// Store recovery codes.
	if err := s.authRepo.StoreRecoveryCodes(ctx, userID, recoveryCodes); err != nil {
		return nil, fmt.Errorf("store recovery codes: %w", err)
	}

	return &Setup2FAResponse{
		Secret:        key.Secret(),
		QRCode:        qrBase64,
		RecoveryCodes: recoveryCodes,
	}, nil
}

func (s *authService) Enable2FA(ctx context.Context, userID string, req Verify2FARequest) error {
	if err := s.validate.Struct(req); err != nil {
		return err
	}

	tfa, err := s.authRepo.Get2FAByUserID(ctx, userID)
	if err != nil {
		return Err2FANotEnabled
	}

	if tfa.IsEnabled {
		return Err2FAAlreadyEnabled
	}

	// Verify the TOTP code to confirm the user has set up their authenticator.
	if !totp.Validate(req.Code, tfa.SecretKey) {
		return ErrInvalid2FACode
	}

	if err := s.authRepo.Enable2FA(ctx, userID); err != nil {
		return fmt.Errorf("enable 2fa: %w", err)
	}

	return nil
}

func (s *authService) Disable2FA(ctx context.Context, userID string, req Verify2FARequest) error {
	if err := s.validate.Struct(req); err != nil {
		return err
	}

	tfa, err := s.authRepo.Get2FAByUserID(ctx, userID)
	if err != nil {
		return err
	}

	if !tfa.IsEnabled {
		return Err2FANotEnabled
	}

	// Verify the code before disabling (TOTP or recovery code).
	if !totp.Validate(req.Code, tfa.SecretKey) {
		if !s.useRecoveryCode(ctx, userID, tfa, req.Code) {
			return ErrInvalid2FACode
		}
	}

	if err := s.authRepo.Delete2FA(ctx, userID); err != nil {
		return fmt.Errorf("disable 2fa: %w", err)
	}

	return nil
}

// useRecoveryCode checks and consumes a recovery code.
func (s *authService) useRecoveryCode(ctx context.Context, userID string, tfa *TwoFactorAuth, code string) bool {
	code = strings.TrimSpace(strings.ToUpper(code))
	remaining := make([]string, 0, len(tfa.RecoveryCodes))
	found := false

	for _, rc := range tfa.RecoveryCodes {
		if strings.ToUpper(rc) == code && !found {
			found = true
			continue
		}
		remaining = append(remaining, rc)
	}

	if found {
		// Fail closed: if we can't persist the consumed set, the code is still
		// valid in the DB, so we must not treat this attempt as a successful
		// single-use. Deny and let the user retry rather than silently allowing
		// the same code to be reused.
		if err := s.authRepo.StoreRecoveryCodes(ctx, userID, remaining); err != nil {
			slog.Error("consume recovery code: persist failed", "user_id", userID, "err", err)
			return false
		}
	}

	return found
}

func generateRecoveryCodes(count int) ([]string, error) {
	codes := make([]string, count)
	for i := 0; i < count; i++ {
		b := make([]byte, 5)
		if _, err := rand.Read(b); err != nil {
			return nil, fmt.Errorf("generate recovery codes: %w", err)
		}
		code := fmt.Sprintf("%X", b)
		// Format as XXXX-XXXX for readability.
		if len(code) >= 8 {
			code = code[:4] + "-" + code[4:8]
		}
		codes[i] = code
	}
	return codes, nil
}

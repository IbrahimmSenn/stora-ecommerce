package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "test-secret-key-for-unit-tests"

func TestGenerateTokenPair(t *testing.T) {
	pair, err := GenerateTokenPair("user-123", "test@example.com", "user", testSecret)
	require.NoError(t, err)
	assert.NotEmpty(t, pair.AccessToken)
	assert.NotEmpty(t, pair.RefreshToken)
	assert.NotEqual(t, pair.AccessToken, pair.RefreshToken)
}

func TestValidateToken_Valid(t *testing.T) {
	pair, err := GenerateTokenPair("user-123", "test@example.com", "admin", testSecret)
	require.NoError(t, err)

	claims, err := ValidateToken(pair.AccessToken, testSecret)
	require.NoError(t, err)
	assert.Equal(t, "user-123", claims.UserID)
	assert.Equal(t, "test@example.com", claims.Email)
	assert.Equal(t, "admin", claims.Role)
}

func TestValidateToken_WrongSecret(t *testing.T) {
	pair, err := GenerateTokenPair("user-123", "test@example.com", "user", testSecret)
	require.NoError(t, err)

	_, err = ValidateToken(pair.AccessToken, "wrong-secret")
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestValidateToken_Expired(t *testing.T) {
	claims := Claims{
		UserID: "user-123",
		Email:  "test@example.com",
		Role:   "user",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	require.NoError(t, err)

	_, err = ValidateToken(token, testSecret)
	assert.ErrorIs(t, err, ErrExpiredToken)
}

func TestValidateToken_MalformedInput(t *testing.T) {
	tests := []struct {
		name  string
		token string
	}{
		{"empty string", ""},
		{"random garbage", "not.a.jwt"},
		{"partial jwt", "eyJhbGciOiJIUzI1NiJ9."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateToken(tt.token, testSecret)
			assert.ErrorIs(t, err, ErrInvalidToken)
		})
	}
}

func TestValidateRefreshToken_Valid(t *testing.T) {
	pair, err := GenerateTokenPair("user-456", "refresh@example.com", "user", testSecret)
	require.NoError(t, err)

	claims, err := ValidateRefreshToken(pair.RefreshToken, testSecret)
	require.NoError(t, err)
	assert.Equal(t, "user-456", claims.Subject)
}

func TestValidateRefreshToken_WrongSecret(t *testing.T) {
	pair, err := GenerateTokenPair("user-456", "refresh@example.com", "user", testSecret)
	require.NoError(t, err)

	_, err = ValidateRefreshToken(pair.RefreshToken, "wrong-secret")
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestValidateToken_UnsupportedSigningMethod(t *testing.T) {
	// Create a token with "none" signing method
	token := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.RegisteredClaims{
		Subject:   "user-123",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
	})
	tokenString, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	_, err = ValidateToken(tokenString, testSecret)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

//go:build integration

package auth

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/IbrahimmSenn/stora-ecommerce/internal/crypto"
	"github.com/IbrahimmSenn/stora-ecommerce/internal/testdb"
)

func seedTokenRow(t *testing.T, pool *pgxpool.Pool) (tokenID string) {
	t.Helper()
	userID := uuid.New()
	hmac := uuid.New().String() // unique blind index stand-in
	_, err := pool.Exec(context.Background(),
		`INSERT INTO users (id, password_hash, email_hmac) VALUES ($1, $2, $3)`,
		userID, "x", []byte(hmac))
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID) })

	id := uuid.New()
	_, err = pool.Exec(context.Background(),
		`INSERT INTO refresh_tokens (id, token, user_id, used, expires_at)
		 VALUES ($1,$2,$3,false,$4)`,
		id, "itest-"+id.String(), userID, time.Now().Add(time.Hour))
	require.NoError(t, err)
	return id.String()
}

// TestIntegration_RefreshRotationRace: two concurrent refreshes present the
// same token. The atomic `AND used = false` guard means exactly one UPDATE
// affects a row — the winner gets nil, the loser gets ErrTokenUsed.
func TestIntegration_RefreshRotationRace(t *testing.T) {
	pool := testdb.Pool(t)
	enc, err := crypto.NewEncryptor("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	require.NoError(t, err)
	repo := NewAuthRepository(pool, enc)

	tokenID := seedTokenRow(t, pool)

	var wg sync.WaitGroup
	var mu sync.Mutex
	var nilCount, usedCount int
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := repo.MarkRefreshTokenUsed(context.Background(), tokenID)
			mu.Lock()
			switch {
			case err == nil:
				nilCount++
			case err == ErrTokenUsed:
				usedCount++
			}
			mu.Unlock()
		}()
	}
	wg.Wait()

	assert.Equal(t, 1, nilCount, "exactly one refresh should win the rotation")
	assert.Equal(t, 1, usedCount, "the loser must be told the token was already used")
}

package user

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedUser(m *mockRepo, role string) string {
	id := uuid.New()
	email := id.String() + "@shop.com"
	m.users[email] = User{Id: id, Email: email, Role: role}
	return id.String()
}

func TestAdminSetRole_RejectsInvalidRole(t *testing.T) {
	svc := NewService(newMockRepo(), 0, nil, nil)
	repo := svc.(*userService).repo.(*mockRepo)
	id := seedUser(repo, "customer")

	err := svc.AdminSetRole(context.Background(), id, "superuser")
	assert.ErrorIs(t, err, ErrInvalidRole)
}

func TestAdminSetRole_Promotes(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, 0, nil, nil)
	id := seedUser(repo, "customer")

	require.NoError(t, svc.AdminSetRole(context.Background(), id, "sales"))
	assert.Equal(t, "sales", repo.users[id+"@shop.com"].Role)
}

func TestAdminSetRole_BlocksRemovingLastAdmin(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, 0, nil, nil)
	id := seedUser(repo, "admin") // the only admin

	err := svc.AdminSetRole(context.Background(), id, "customer")
	assert.ErrorIs(t, err, ErrLastAdmin)
}

func TestAdminSetRole_AllowsDemotionWhenAnotherAdminExists(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, 0, nil, nil)
	id := seedUser(repo, "admin")
	seedUser(repo, "admin") // a second admin

	require.NoError(t, svc.AdminSetRole(context.Background(), id, "support"))
}

func TestAdminListUsers_ReturnsAll(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, 0, nil, nil)
	seedUser(repo, "customer")
	seedUser(repo, "admin")

	list, err := svc.AdminListUsers(context.Background(), 1, 25)
	require.NoError(t, err)
	assert.Equal(t, 2, list.Total)
}

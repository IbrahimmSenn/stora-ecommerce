// admin.go — admin user management: listing accounts and assigning roles.
package user

import (
	"context"
)

var validRoles = map[string]bool{
	"admin":    true,
	"support":  true,
	"sales":    true,
	"customer": true,
}

func (s *userService) AdminListUsers(ctx context.Context, page, pageSize int) (*AdminUserList, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 25
	}

	users, total, err := s.repo.ListAll(ctx, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, err
	}

	out := make([]AdminUser, 0, len(users))
	for _, u := range users {
		out = append(out, AdminUser{Id: u.Id, Email: u.Email, Role: u.Role, CreatedAt: u.CreatedAt})
	}
	return &AdminUserList{Users: out, Total: total, Page: page, PageSize: pageSize}, nil
}

// AdminSetRole assigns a role to a user. Guards against removing the platform's
// last admin so the dashboard can't lock everyone out.
func (s *userService) AdminSetRole(ctx context.Context, userID, role string) error {
	if !validRoles[role] {
		return ErrInvalidRole
	}

	current, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if current.Role == "admin" && role != "admin" {
		admins, err := s.repo.CountByRole(ctx, "admin")
		if err != nil {
			return err
		}
		if admins <= 1 {
			return ErrLastAdmin
		}
	}

	return s.repo.UpdateRole(ctx, userID, role)
}

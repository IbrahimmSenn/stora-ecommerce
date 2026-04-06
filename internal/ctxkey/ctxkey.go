// ctxkey.go — shared context keys for passing user identity between middleware and handlers.
// Lives in its own package to avoid import cycles between auth and middleware.
package ctxkey

type Key string

const (
	UserID Key = "userID"
	Email  Key = "email"
	Role   Key = "role"
)

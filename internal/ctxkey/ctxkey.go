// ctxkey.go — context keys shared across packages to avoid import cycles.
package ctxkey

type Key string

const (
	UserID Key = "userID"
	Email  Key = "email"
	Role   Key = "role"
)

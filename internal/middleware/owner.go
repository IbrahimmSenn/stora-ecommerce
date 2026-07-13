package middleware

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/IbrahimmSenn/stora-ecommerce/internal/ctxkey"
)

// ResolveOwner extracts the owning identity of a request: the authenticated
// user ID from the JWT context, or the guest session ID from the cookie.
// Authenticated users take priority. Both nil means an anonymous request with
// no guest session. Shared by the cart, orders, payments, product, and
// recommend handlers so they resolve ownership identically.
func ResolveOwner(r *http.Request) (userID, guestID *uuid.UUID) {
	if raw, ok := r.Context().Value(ctxkey.UserID).(string); ok && raw != "" {
		if uid, err := uuid.Parse(raw); err == nil {
			return &uid, nil
		}
	}

	if c, err := r.Cookie(GuestSessionCookie); err == nil {
		if gid, err := uuid.Parse(c.Value); err == nil {
			return nil, &gid
		}
	}

	return nil, nil
}

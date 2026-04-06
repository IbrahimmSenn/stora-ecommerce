// provider.go — common interface that all OAuth providers (Google, Facebook) implement.
package oauth

// UserInfo is the standardized user data returned by any OAuth provider.
type UserInfo struct {
	ProviderUserID string
	Email          string
	Provider       string // "google" or "facebook"
}

// Provider is the interface for all OAuth providers.
// This makes it trivial to add new providers without changing service logic.
type Provider interface {
	// Name returns the provider identifier (e.g., "google", "facebook").
	Name() string
	// AuthURL returns the URL to redirect the user to for authentication.
	AuthURL(state string) string
	// Exchange takes the OAuth callback code and returns the user's info.
	Exchange(code string) (*UserInfo, error)
}

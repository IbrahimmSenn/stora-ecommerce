// google.go — Google OAuth provider: consent redirect and token exchange.
package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const googleUserInfoURL = "https://www.googleapis.com/oauth2/v2/userinfo"

type googleProvider struct {
	config *oauth2.Config
}

// NewGoogle creates a Google OAuth provider.
func NewGoogle(clientID, clientSecret, redirectURL string) Provider {
	return &googleProvider{
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       []string{"email", "profile"},
			Endpoint:     google.Endpoint,
		},
	}
}

func (g *googleProvider) Name() string { return "google" }

func (g *googleProvider) AuthURL(state string) string {
	return g.config.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

func (g *googleProvider) Exchange(code string) (*UserInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	token, err := g.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("google exchange: %w", err)
	}

	client := g.config.Client(ctx, token)
	resp, err := client.Get(googleUserInfoURL)
	if err != nil {
		return nil, fmt.Errorf("google userinfo: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("google read body: %w", err)
	}

	var info struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	}
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("google decode: %w", err)
	}

	if info.Email == "" {
		return nil, fmt.Errorf("google: email not available")
	}

	return &UserInfo{
		ProviderUserID: info.ID,
		Email:          info.Email,
		Provider:       "google",
	}, nil
}

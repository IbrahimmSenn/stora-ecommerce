package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/facebook"
)

const facebookUserInfoURL = "https://graph.facebook.com/me?fields=id,email"

type facebookProvider struct {
	config *oauth2.Config
}

// NewFacebook creates a Facebook OAuth provider.
func NewFacebook(clientID, clientSecret, redirectURL string) Provider {
	return &facebookProvider{
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       []string{"email"},
			Endpoint:     facebook.Endpoint,
		},
	}
}

func (f *facebookProvider) Name() string { return "facebook" }

func (f *facebookProvider) AuthURL(state string) string {
	return f.config.AuthCodeURL(state)
}

func (f *facebookProvider) Exchange(code string) (*UserInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	token, err := f.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("facebook exchange: %w", err)
	}

	client := f.config.Client(ctx, token)
	resp, err := client.Get(facebookUserInfoURL)
	if err != nil {
		return nil, fmt.Errorf("facebook userinfo: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("facebook read body: %w", err)
	}

	var info struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	}
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("facebook decode: %w", err)
	}

	if info.Email == "" {
		return nil, fmt.Errorf("facebook: email not available")
	}

	return &UserInfo{
		ProviderUserID: info.ID,
		Email:          info.Email,
		Provider:       "facebook",
	}, nil
}

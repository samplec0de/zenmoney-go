package zenmoney

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const (
	authURL  = "https://api.zenmoney.ru/oauth2/authorize/"
	tokenURL = "https://api.zenmoney.ru/oauth2/token/"
)

// Token holds OAuth2 credentials returned by ZenMoney.
type Token struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

// OAuthConfig holds the application credentials for OAuth2.
type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
}

// AuthCodeURL returns the URL to redirect the user to for authorization.
func (c *OAuthConfig) AuthCodeURL() string {
	v := url.Values{
		"response_type": {"code"},
		"client_id":     {c.ClientID},
		"redirect_uri":  {c.RedirectURI},
	}
	return authURL + "?" + v.Encode()
}

// Exchange trades an authorization code for an access token.
func (c *OAuthConfig) Exchange(code string) (*Token, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {c.ClientID},
		"client_secret": {c.ClientSecret},
		"code":          {code},
		"redirect_uri":  {c.RedirectURI},
	}

	resp, err := http.Post(tokenURL, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("token exchange request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange: status %d", resp.StatusCode)
	}

	var tok Token
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return nil, fmt.Errorf("token decode: %w", err)
	}
	return &tok, nil
}

// Refresh obtains a new access token using a refresh token.
func (c *OAuthConfig) Refresh(refreshToken string) (*Token, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {c.ClientID},
		"client_secret": {c.ClientSecret},
		"refresh_token": {refreshToken},
	}

	resp, err := http.Post(tokenURL, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("token refresh request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token refresh: status %d", resp.StatusCode)
	}

	var tok Token
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return nil, fmt.Errorf("token decode: %w", err)
	}
	return &tok, nil
}

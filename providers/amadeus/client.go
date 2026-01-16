package amadeus

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const (
	BaseURLTest       = "https://test.api.amadeus.com"
	BaseURLProduction = "https://api.amadeus.com"
)

// Client is the main Amadeus API client
type Client struct {
	ClientID     string
	ClientSecret string
	BaseURL      string
	HTTPClient   *http.Client
	Token        *AuthToken
}

// AuthToken represents the OAuth2 token response
type AuthToken struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
	Expiry      time.Time
}

// NewClient creates a new Amadeus client
// Returns an error if the client cannot be initialized
func NewClient(clientID, clientSecret string, isProduction bool) (*Client, error) {
	if clientID == "" {
		return nil, fmt.Errorf("client ID is required")
	}
	if clientSecret == "" {
		return nil, fmt.Errorf("client secret is required")
	}

	baseURL := BaseURLTest
	if isProduction {
		baseURL = BaseURLProduction
	}

	return &Client{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		BaseURL:      baseURL,
		HTTPClient:   &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// Authenticate obtains a new access token
func (c *Client) Authenticate() error {
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", c.ClientID)
	data.Set("client_secret", c.ClientSecret)

	req, err := http.NewRequest("POST", c.BaseURL+"/v1/security/oauth2/token", bytes.NewBufferString(data.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("authentication failed: %s", resp.Status)
	}

	var token AuthToken
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return err
	}

	// Set expiry time (subtract 10 seconds for buffer)
	token.Expiry = time.Now().Add(time.Duration(token.ExpiresIn)*time.Second - 10*time.Second)
	c.Token = &token

	return nil
}

// doRequest performs an authenticated HTTP request
func (c *Client) doRequest(method, endpoint string, body interface{}) (*http.Response, error) {
	if c.Token == nil || time.Now().After(c.Token.Expiry) {
		if err := c.Authenticate(); err != nil {
			return nil, fmt.Errorf("failed to refresh token: %w", err)
		}
	}

	var reqBody []byte
	var err error
	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}

	url := c.BaseURL + endpoint
	req, err := http.NewRequest(method, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.Token.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	return c.HTTPClient.Do(req)
}

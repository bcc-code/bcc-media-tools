package main

import (
	"fmt"
	"github.com/go-resty/resty/v2"
	"os"
	"time"
)

type BMMApi struct {
	client *resty.Client
	token  *BMMToken
}

func getToken(tokenBaseURL, clientID, clientSecret, audience string) (*BMMToken, error) {
	if os.Getenv("BMM_DEBUG_TOKEN") != "" {
		fmt.Printf("WARNING: Using DEBUG token. Expired token will not be automatically refreshed\n")
		return &BMMToken{
			AccessToken: os.Getenv("BMM_DEBUG_TOKEN"),
			ExpiresIn:   24 * 60 * 60,
			CreatedAt:   time.Now(),
		}, nil
	}

	r := resty.New()
	r.SetBaseURL(tokenBaseURL)
	res, err := r.R().SetBody(map[string]string{
		"client_id":     clientID,
		"client_secret": clientSecret,
		"audience":      audience,
		"grant_type":    "client_credentials",
	}).SetResult(&BMMToken{}).Post("oauth/token")

	if err != nil {
		return nil, err
	}

	token := res.Result().(*BMMToken)

	if token.AccessToken == "" {
		return nil, fmt.Errorf("faulure to get token: %s", res.String())
	}

	token.CreatedAt = time.Now()

	return token, nil
}

func NewBMMToken(tokenBaseURL, clientID, clientSecret, audience string) (*BMMToken, error) {
	t, err := getToken(tokenBaseURL, clientID, clientSecret, audience)
	if err != nil {
		return nil, err
	}

	t.tokenBaseURL = tokenBaseURL
	t.clientID = clientID
	t.clientSecret = clientSecret
	t.audience = audience

	return t, nil
}

type BMMToken struct {
	AccessToken string `json:"access_token"`
	Scope       string `json:"scope"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
	CreatedAt   time.Time

	tokenBaseURL string
	clientID     string
	clientSecret string
	audience     string
}

func (t *BMMToken) GetAccessToken() string {
	if t.Expired() {
		err := t.Refresh()
		if err != nil {
			// TODO: Maybe not panic?
			panic(err)
		}
	}

	return t.AccessToken
}

func (t *BMMToken) Expired() bool {
	return time.Since(t.CreatedAt)+10*time.Second > time.Duration(int64(t.ExpiresIn)*int64(time.Second))
}

func (t *BMMToken) Refresh() error {
	newToken, err := getToken(t.tokenBaseURL, t.clientID, t.clientSecret, t.audience)
	if err != nil {
		return err
	}

	t.AccessToken = newToken.AccessToken
	t.Scope = newToken.Scope
	t.ExpiresIn = newToken.ExpiresIn
	t.TokenType = newToken.TokenType
	t.CreatedAt = time.Now()

	return nil
}

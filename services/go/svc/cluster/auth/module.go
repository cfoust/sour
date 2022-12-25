package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const (
	API_ENDPOINT = "https://discord.com/api/v10"
)

type DiscordService struct {
	clientId    string
	secret      string
	redirectURI string
}

func NewDiscordService(clientId string, secret string, redirectURI string) *DiscordService {
	return &DiscordService{
		clientId:    clientId,
		secret:      secret,
		redirectURI: redirectURI,
	}
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string
}

func (d *DiscordService) GetAccessToken(code string) (*TokenResponse, error) {
	v := url.Values{}
	v.Set("client_id", d.clientId)
	v.Set("client_secret", d.secret)
	v.Set("grant_type", "authorization_code")
	v.Set("code", code)
	v.Set("redirect_uri", d.redirectURI)

	resp, err := http.PostForm(
		fmt.Sprintf("%s/oauth2/token", API_ENDPOINT),
		v,
	)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %s", resp.Status)
	}

	buffer, err := io.ReadAll(resp.Body)

	var token TokenResponse
	err = json.Unmarshal(buffer, &token)
	if err != nil {
		return nil, err
	}

	return &token, nil
}

type DiscordUser struct {
	Id            string
	Username      string
	Discriminator string
	Avatar        string
}

func (d *DiscordService) GetUser(token string) (*DiscordUser, error) {
	client := http.Client{}
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/users/@me", API_ENDPOINT),
		nil,
	)
	if err != nil {
		return nil, err
	}

	req.Header = http.Header{"Authorization": {fmt.Sprintf("Bearer %s", token)}}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %s", resp.Status)
	}

	buffer, err := io.ReadAll(resp.Body)

	var user DiscordUser
	err = json.Unmarshal(buffer, &user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"unsafe"

	"github.com/cfoust/sour/svc/cluster/auth/crypto"
	"github.com/cfoust/sour/svc/cluster/config"
	"github.com/cfoust/sour/svc/cluster/state"
)

const (
	API_ENDPOINT = "https://discord.com/api/v10"
)

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string
}

type DiscordUser struct {
	Id            string
	Username      string
	Discriminator string
	Avatar        string
}

type KeyPair struct {
	Public  string
	Private string
}

type User struct {
	Discord DiscordUser
	Keys    KeyPair
}

func GenerateSauerKey(seed string) (public string, private string) {
	priv := make([]byte, 120)
	pub := make([]byte, 120)
	crypto.Genauthkey(seed, uintptr(unsafe.Pointer(&priv[0])), uintptr(unsafe.Pointer(&pub[0])))
	return string(pub), string(priv)
}

func GenerateAuthKey() (KeyPair, error) {
	number, err := rand.Int(rand.Reader, big.NewInt(1073741824))
	if err != nil {
		return KeyPair{}, err
	}
	bytes := sha256.Sum256([]byte(fmt.Sprintf("%d", number)))
	hash := fmt.Sprintf("%x", bytes)[:24]
	public, private := GenerateSauerKey(hash)
	return KeyPair{public, private}, nil
}

type DiscordService struct {
	clientId    string
	secret      string
	redirectURI string
	state       *state.StateService
}

func NewDiscordService(config config.DiscordSettings, state *state.StateService) *DiscordService {
	return &DiscordService{
		clientId:    config.Id,
		secret:      config.Secret,
		redirectURI: config.RedirectURI,
		state:       state,
	}
}

func (d *DiscordService) FetchAccessToken(code string) (*TokenResponse, error) {
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

func (d *DiscordService) RefreshAccessToken(refreshToken string) (*TokenResponse, error) {
	v := url.Values{}
	v.Set("client_id", d.clientId)
	v.Set("client_secret", d.secret)
	v.Set("grant_type", "refresh_token")
	v.Set("refresh_token", refreshToken)

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

func (d *DiscordService) SaveTokenBundle(ctx context.Context, bundle *TokenResponse) error {
	return d.state.SaveToken(
		ctx,
		bundle.AccessToken,
		bundle.ExpiresIn,
		bundle.RefreshToken,
	)
}

// Get or create an auth key for a user to log in on desktop Sauerbraten.
func (d *DiscordService) GetAuthKey(ctx context.Context, id string) (KeyPair, error) {
	public, private, err := d.state.GetAuthKeyForUser(ctx, id)

	pair := KeyPair{
		Public:  public,
		Private: private,
	}

	if err != nil && err != state.Nil {
		return pair, err
	}

	if err == nil {
		return pair, nil
	}

	// Generate one
	pair, err = GenerateAuthKey()
	if err != nil {
		return pair, err
	}

	err = d.state.SaveAuthKeyForUser(ctx, id, pair.Public, pair.Private)
	if err != nil {
		return pair, err
	}

	return pair, nil
}

func (d *DiscordService) AuthenticateCode(ctx context.Context, code string) (*User, error) {
	token, err := d.state.GetTokenForCode(ctx, code)

	if err != nil && err != state.Nil {
		return nil, err
	}

	// Attempt to fetch
	if err == state.Nil {
		bundle, err := d.FetchAccessToken(code)
		if err != nil {
			return nil, err
		}

		err = d.state.SaveTokenForCode(
			ctx,
			code,
			bundle.AccessToken,
			bundle.ExpiresIn,
		)
		if err != nil {
			return nil, err
		}

		err = d.SaveTokenBundle(
			ctx,
			bundle,
		)
		if err != nil {
			return nil, err
		}

		token = bundle.AccessToken
	}

	needsRefresh, err := d.state.TokenNeedsRefresh(
		ctx,
		token,
	)
	if err != nil {
		return nil, err
	}

	if needsRefresh {
		refresh, err := d.state.GetRefreshForToken(
			ctx,
			token,
		)
		if err != nil {
			return nil, err
		}

		bundle, err := d.RefreshAccessToken(refresh)
		if err != nil {
			return nil, err
		}

		err = d.state.SaveTokenForCode(
			ctx,
			code,
			bundle.AccessToken,
			bundle.ExpiresIn,
		)
		if err != nil {
			return nil, err
		}

		err = d.SaveTokenBundle(
			ctx,
			bundle,
		)
		if err != nil {
			return nil, err
		}

		token = bundle.AccessToken
	}

	discordUser, err := d.GetUser(token)
	if err != nil {
		return nil, err
	}

	pair, err := d.GetAuthKey(ctx, discordUser.Id)
	if err != nil {
		return nil, err
	}

	return &User{Discord: *discordUser, Keys: pair}, err
}

package auth

import (
	"bytes"
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

func (u *DiscordUser) Reference() string {
	return fmt.Sprintf("%s#%s", u.Username, u.Discriminator)
}

type KeyPair struct {
	Public  string
	Private string
}

type User struct {
	Discord DiscordUser
	Keys    KeyPair
}

func GenerateSeed() (string, error) {
	number, err := rand.Int(rand.Reader, big.NewInt(1073741824))
	if err != nil {
		return "", err
	}
	bytes := sha256.Sum256([]byte(fmt.Sprintf("%d", number)))
	return fmt.Sprintf("%x", bytes)[:24], nil
}

type Challenge struct {
	// Discord Id
	Id           string
	Question     string
	Answer       uintptr
	AnswerString string
}

func (c *Challenge) Destroy() {
	crypto.Freechallenge(c.Answer)
}

func (c *Challenge) Check(answer string) bool {
	return crypto.Checkchallenge(answer, c.Answer)
}

func CleanString(data []byte) string {
	end := bytes.IndexByte(data[:], 0)
	return string(data[:end])
}

func GenerateChallenge(id string, publicKey string) (*Challenge, error) {
	seed, err := GenerateSeed()
	if err != nil {
		return nil, err
	}
	challengeString := make([]byte, 120)
	answerString := make([]byte, 120)
	ptr := crypto.Genchallenge(
		publicKey,
		seed,
		len(seed),
		uintptr(unsafe.Pointer(&challengeString[0])),
		uintptr(unsafe.Pointer(&answerString[0])),
	)
	return &Challenge{
		Id:           id,
		Question:     CleanString(challengeString),
		AnswerString: CleanString(answerString),
		Answer:       ptr,
	}, nil
}

func GenerateSauerKey(seed string) (public string, private string) {
	priv := make([]byte, 120)
	pub := make([]byte, 120)
	crypto.Genauthkey(seed, uintptr(unsafe.Pointer(&priv[0])), uintptr(unsafe.Pointer(&pub[0])))

	return CleanString(pub), CleanString(priv)
}

func GenerateAuthKey() (*KeyPair, error) {
	seed, err := GenerateSeed()
	if err != nil {
		return nil, err
	}
	public, private := GenerateSauerKey(seed)
	return &KeyPair{public, private}, nil
}

type DiscordService struct {
	clientId    string
	secret      string
	redirectURI string
	State       *state.StateService
}

func NewDiscordService(config config.DiscordSettings, state *state.StateService) *DiscordService {
	return &DiscordService{
		clientId:    config.Id,
		secret:      config.Secret,
		redirectURI: config.RedirectURI,
		State:       state,
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
	return d.State.SaveToken(
		ctx,
		bundle.AccessToken,
		bundle.ExpiresIn,
		bundle.RefreshToken,
	)
}

// Get or create an auth key for a user to log in on desktop Sauerbraten.
func (d *DiscordService) GetAuthKey(ctx context.Context, id string) (*KeyPair, error) {
	public, private, err := d.State.GetAuthKeyForId(ctx, id)

	existing := KeyPair{
		Public:  public,
		Private: private,
	}

	if err != nil && err != state.Nil {
		return &existing, err
	}

	if err == nil {
		return &existing, nil
	}

	return nil, state.Nil
}

func (d *DiscordService) EnsureAuthKey(ctx context.Context, id string) (*KeyPair, error) {
	pair, err := d.GetAuthKey(ctx, id)

	if err == nil {
		return pair, nil
	}

	if err != state.Nil && err != nil {
		return nil, err
	}

	// Generate one
	pair, err = GenerateAuthKey()
	if err != nil {
		return pair, err
	}

	err = d.State.SaveAuthKeyForUser(ctx, id, pair.Public, pair.Private)
	if err != nil {
		return pair, err
	}

	return pair, nil
}

func (d *DiscordService) CheckRefreshToken(ctx context.Context, token string) error {
	needsRefresh, err := d.State.TokenNeedsRefresh(
		ctx,
		token,
	)
	if err != nil {
		return err
	}

	if !needsRefresh {
		return nil
	}

	refresh, err := d.State.GetRefreshForToken(
		ctx,
		token,
	)
	if err != nil {
		return err
	}

	bundle, err := d.RefreshAccessToken(refresh)
	if err != nil {
		return err
	}

	user, err := d.GetUser(bundle.AccessToken)
	if err != nil {
		return err
	}

	err = d.State.SetTokenForId(
		ctx,
		user.Id,
		bundle.AccessToken,
		bundle.ExpiresIn,
	)
	if err != nil {
		return err
	}

	err = d.SaveTokenBundle(
		ctx,
		bundle,
	)
	if err != nil {
		return err
	}

	return nil
}

func (d *DiscordService) FetchUser(ctx context.Context, token string) (*User, error) {
	discordUser, err := d.GetUser(token)
	if err != nil {
		return nil, err
	}

	pair, err := d.EnsureAuthKey(ctx, discordUser.Id)
	if err != nil {
		return nil, err
	}

	return &User{Discord: *discordUser, Keys: *pair}, err
}

func (d *DiscordService) AuthenticateCode(ctx context.Context, code string) (*User, error) {
	token, err := d.State.GetTokenForCode(ctx, code)

	if err == nil {
		err = d.CheckRefreshToken(ctx, token)
		if err != nil {
			return nil, err
		}

		return d.FetchUser(ctx, token)
	}

	if err != nil && err != state.Nil {
		return nil, err
	}

	// Can only be state.Nil, fetch the token for this
	bundle, err := d.FetchAccessToken(code)
	if err != nil {
		return nil, err
	}

	// Need to fetch the user to get their ID
	discordUser, err := d.GetUser(bundle.AccessToken)
	if err != nil {
		return nil, err
	}

	err = d.State.SetIdForCode(
		ctx,
		code,
		discordUser.Id,
		bundle.ExpiresIn,
	)
	if err != nil {
		return nil, err
	}

	err = d.State.SetTokenForId(
		ctx,
		discordUser.Id,
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

	return d.FetchUser(ctx, bundle.AccessToken)
}

func (d *DiscordService) AuthenticateId(ctx context.Context, id string) (*User, error) {
	token, err := d.State.GetTokenForId(
		ctx,
		id,
	)
	if err != nil {
		return nil, err
	}

	err = d.CheckRefreshToken(ctx, token)
	if err != nil {
		return nil, err
	}

	return d.FetchUser(ctx, token)
}

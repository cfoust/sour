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
	"time"
	"unsafe"

	"github.com/cfoust/sour/pkg/server/auth/crypto"
	"github.com/cfoust/sour/pkg/server/config"
	"github.com/cfoust/sour/pkg/server/state"

	"gorm.io/gorm"
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
	db          *gorm.DB
}

func NewDiscordService(config config.DiscordSettings, state *state.StateService, db *gorm.DB) *DiscordService {
	return &DiscordService{
		clientId:    config.Id,
		secret:      config.Secret,
		redirectURI: config.RedirectURI,
		State:       state,
		db:          db,
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

func applyDiscord(user *state.User, discord *DiscordUser) {
	user.Username = discord.Username
	user.Discriminator = discord.Discriminator
	user.Avatar = discord.Avatar
}

func (d *DiscordService) updateLogin(ctx context.Context, user *state.User) error {
	return d.db.WithContext(ctx).Save(user).Error
}

var LoginExpired = fmt.Errorf("session expired")

func createAuthCode(ctx context.Context, db *gorm.DB, user *state.User, code string) error {
	authCode := state.AuthCode{
		UserID:  user.ID,
		Value:   code,
		Expires: time.Now().Add(30 * time.Hour * 24),
	}
	return db.Create(&authCode).Error
}

func (d *DiscordService) AuthenticateCode(ctx context.Context, code string) (*state.User, error) {
	db := d.db.WithContext(ctx)

	// First check the database
	authCode := state.AuthCode{}
	err := db.Where(state.AuthCode{
		Value: code,
	}).Joins("User").First(&authCode).Error

	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// The user is already associated with this code
	if err == nil {
		if authCode.Expires.Before(time.Now()) {
			err = db.Delete(&authCode).Error
			if err != nil {
				return nil, err
			}
			return nil, LoginExpired
		}

		user := authCode.User
		user.LastLogin = time.Now()
		err = db.Save(user).Error
		if err != nil {
			return nil, err
		}

		return user, nil
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

	var user state.User
	err = db.Where(
		state.User{
			UUID: discordUser.Id,
		},
	).
		First(&user).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// The user already existed, they just have a new code
	if err == nil {
		err = createAuthCode(ctx, db, &user, code)
		if err != nil {
			return nil, err
		}

		applyDiscord(&user, discordUser)
		user.LastLogin = time.Now()
		err = db.Save(&user).Error
		if err != nil {
			return nil, err
		}

		return &user, nil
	}

	// Create the user
	pair, err := GenerateAuthKey()
	if err != nil {
		return nil, err
	}

	user = state.User{
		Nickname:   "unnamed",
		UUID:       discordUser.Id,
		LastLogin:  time.Now(),
		PublicKey:  pair.Public,
		PrivateKey: pair.Private,
	}

	applyDiscord(&user, discordUser)

	err = db.Create(&user).Error
	if err != nil {
		return nil, err
	}

	err = createAuthCode(ctx, db, &user, code)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (d *DiscordService) AuthenticateId(ctx context.Context, id string) (*state.User, error) {
	db := d.db.WithContext(ctx)

	user := state.User{}
	err := db.Where(
		state.User{
			UUID: id,
		},
	).
		First(&user).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	if err == gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("user not found for id %s", id)
	}

	user.LastLogin = time.Now()
	err = db.Save(user).Error
	if err != nil {
		return nil, err
	}

	return &user, nil
}

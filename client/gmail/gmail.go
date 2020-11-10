package gmail

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"net/url"
)

const (
	OAUTH2_SCOPE            = "https://mail.google.com/ email"
	OAUTH2_REDIRECT_URI_CLI = "urn:ietf:wg:oauth:2.0:oob"

	OAUTH2_AUTH_BASE_URL  = "https://accounts.google.com/o/oauth2/auth"
	OAUTH2_TOKEN_BASE_URL = "https://accounts.google.com/o/oauth2/token"

	GMAIL_INFO_URL = "https://www.googleapis.com/userinfo/email"
)

type Gmail struct {
}

func New() *Gmail {
	return &Gmail{}
}

func (g *Gmail) GetAuthURI(clientID, scope, redirectURI string) string {
	form := url.Values{}
	form.Add("client_id", clientID)
	form.Add("scope", scope)
	form.Add("redirect_uri", redirectURI)
	form.Add("response_type", "code")
	return fmt.Sprintf("%s?%s", OAUTH2_AUTH_BASE_URL, form.Encode())
}

func (g *Gmail) FetchRefreshToken(clientID, clientSecret, authCode, redirectURI string) (string, error) {
	form := url.Values{}
	form.Add("client_id", clientID)
	form.Add("client_secret", clientSecret)
	form.Add("code", authCode)
	form.Add("redirect_uri", redirectURI)
	form.Add("grant_type", "authorization_code")

	resp, err := http.PostForm(OAUTH2_TOKEN_BASE_URL, form)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)
	t := OAuth2AuthedTokens{}
	err = dec.Decode(&t)
	if err == io.EOF {
		return "", fmt.Errorf("auth response from the server is empty")
	} else if err != nil {
		return "", err
	}
	return t.RefreshToken, nil
}

func (g *Gmail) FetchAccessToken(clientID, clientSecret, refreshToken string) (string, error) {
	form := url.Values{}
	form.Add("client_id", clientID)
	form.Add("client_secret", clientSecret)
	form.Add("refresh_token", refreshToken)
	form.Add("grant_type", "refresh_token")
	resp, err := http.PostForm(OAUTH2_TOKEN_BASE_URL, form)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)
	t := OAuth2AuthedTokens{}
	err = dec.Decode(&t)
	if err == io.EOF {
		return "", fmt.Errorf("auth response from the server is empty")
	} else if err != nil {
		return "", err
	}

	return t.AccessToken, nil
}

type OAuth2AuthedTokens struct {
	RefreshToken string `json:"refresh_token"`
	AccessToken  string `json:"access_token"`
}

func (g *Gmail) EmailAddress(accessToken string) (string, error) {
	form := url.Values{}
	form.Add("access_token", accessToken)
	form.Add("alt", "json")
	inforesp, err := http.Get(fmt.Sprintf("%s?%s", GMAIL_INFO_URL, form.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to get email address: %v", err)
	}
	defer inforesp.Body.Close()

	dec := json.NewDecoder(inforesp.Body)
	e := OAuth2Email{}
	err = dec.Decode(&e)
	if err == io.EOF {
		return "", fmt.Errorf("auth response from the server is empty")
	} else if err != nil {
		return "", err
	}
	return e.Data.Email, nil
}

type OAuth2Email struct {
	Data struct {
		Email      string `json:"email"`
		IsVerified bool   `json:"isVerified"`
	} `json:"data"`
}

type xOAuth2Auth struct {
	User, AccessToken string
}

func XOAuth2Auth(user, accessToken string) smtp.Auth {
	return &xOAuth2Auth{User: user, AccessToken: accessToken}
}

func (a *xOAuth2Auth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	data := fmt.Sprintf("user=%s\001auth=Bearer %s\001\001", a.User, a.AccessToken)
	am := base64.StdEncoding.EncodeToString([]byte(data))

	return fmt.Sprintf("XOAUTH2 %s", am), nil, nil
}
func (a *xOAuth2Auth) Next(fromServer []byte, more bool) ([]byte, error) {
	return nil, nil
}

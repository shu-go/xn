package pushbullet

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const (
	OAUTH2_AUTH_BASE_URL  = "https://www.pushbullet.com/authorize"
	OAUTH2_TOKEN_BASE_URL = "https://api.pushbullet.com/oauth2/token"
)

type Pushbullet struct {
}

func New() *Pushbullet {
	return &Pushbullet{}
}

func (g *Pushbullet) GetAuthURI(clientID, redirectURI string) string {
	form := url.Values{}
	form.Add("client_id", clientID)
	form.Add("redirect_uri", redirectURI)
	form.Add("response_type", "code")
	return fmt.Sprintf("%s?%s", OAUTH2_AUTH_BASE_URL, form.Encode())
}

func (g *Pushbullet) FetchAccessToken(clientID, clientSecret, authCode string) (string, error) {
	form := url.Values{}
	form.Add("grant_type", "authorization_code")
	form.Add("client_id", clientID)
	form.Add("client_secret", clientSecret)
	form.Add("code", authCode)

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
	AccessToken string `json:"access_token"`
}

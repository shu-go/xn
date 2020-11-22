package slack

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const (
	oauth2Scope = "chat:write:bot channels:read"

	oauth2AuthBaseURL  = "https://slack.com/oauth/authorize"
	oauth2TokenBaseURL = "https://slack.com/api/oauth.access"
)

type Slack struct {
}

func New() *Slack {
	return &Slack{}
}

func (g *Slack) GetAuthURI(clientID, redirectURI string, optTeamAndState ...string) string {
	form := url.Values{}
	form.Add("client_id", clientID)
	form.Add("scope", oauth2Scope)
	form.Add("redirect_uri", redirectURI)
	if len(optTeamAndState) >= 1 {
		form.Add("team", optTeamAndState[0])
	}
	if len(optTeamAndState) >= 2 {
		form.Add("state", optTeamAndState[1])
	}
	return fmt.Sprintf("%s?%s", oauth2AuthBaseURL, form.Encode())
}

func (g *Slack) FetchAccessToken(clientID, clientSecret, authCode, redirectURI string) (string, error) {
	form := url.Values{}
	form.Add("client_id", clientID)
	form.Add("client_secret", clientSecret)
	form.Add("code", authCode)
	form.Add("redirect_uri", redirectURI)

	resp, err := http.PostForm(oauth2TokenBaseURL, form)
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

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/andrew-d/go-termutil"
	api "github.com/mitsuse/pushbullet-go"
	req "github.com/mitsuse/pushbullet-go/requests"
	"github.com/pkg/browser"

	"github.com/shu-go/xn/client/pushbullet"
	"github.com/shu-go/xn/minredir"
)

var (
	pushbulletOAuth2ClientID     string = ""
	pushbulletOAuth2ClientSecret string = ""
)

type pbCmd struct {
	_ struct{} `help:"notify by Pushbullet"`

	Send pbSendCmd `help:"send a notification"`
	Auth pbAuthCmd
}

type pbSendCmd struct {
	Title string `default:"xn" help:"title"`
	Body  string `help:"body"`
}

type pbAuthCmd struct {
	_       struct{} `help:"authenticate"   usage:"1. go to https://www.pushbullet.com/#settings/clients\n2. make a new OAuth Client\n3. xn pushbullet auth CLIENT_ID CLIENT_SECRET"`
	Port    int      `cli:"port=PORT" default:"7878" help:"a temporal PORT for OAuth authentication."`
	Timeout int      `cli:"timeout=TIMEOUT" default:"60" help:"set TIMEOUT (in seconds) on authentication transaction. < 0 is infinite."`
}

func (c pbSendCmd) Run(global globalCmd, args []string) error {
	config, _ := loadConfig(global.Config)

	if config.Pushbullet.AccessToken == "" {
		return fmt.Errorf("auth first")
	}

	//
	// prepare
	//

	for _, v := range args {
		if len(c.Body) > 0 {
			c.Body += "\n"
		}
		c.Body += v
	}

	if !termutil.Isatty(os.Stdin.Fd()) {
		bytes, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			bytes = []byte{}
		}
		if len(c.Body) == 0 {
			c.Body = string(bytes)
		} else if len(bytes) != 0 {
			c.Body += "\n" + string(bytes)
		}
	}

	if len(c.Body) == 0 {
		return nil
	}

	pb := api.New(config.Pushbullet.AccessToken)
	n := req.NewNote()
	n.Title = c.Title
	n.Body = c.Body
	if _, err := pb.PostPushesNote(n); err != nil {
		return err
	}

	return nil
}

func (c pbAuthCmd) Run(global globalCmd, args []string) error {
	config, _ := loadConfig(global.Config)

	var argClientID, argCLientSecret string
	if len(args) >= 2 {
		argClientID = args[0]
		argCLientSecret = args[1]
	}

	//
	// prepare
	//
	pushbulletOAuth2ClientID = firstNonEmpty(
		argClientID,
		config.Pushbullet.ClientID,
		os.Getenv("PUSHBULLET_OAUTH2_CLIENT_ID"),
		pushbulletOAuth2ClientID)
	pushbulletOAuth2ClientSecret = firstNonEmpty(
		argCLientSecret,
		config.Pushbullet.ClientSecret,
		os.Getenv("PUSHBULLET_OAUTH2_CLIENT_SECRET"),
		pushbulletOAuth2ClientSecret)

	if pushbulletOAuth2ClientID == "" || pushbulletOAuth2ClientSecret == "" {
		fmt.Fprintf(os.Stderr, "both PUSHBULLET_OAUTH2_CLIENT_ID and PUSHBULLET_OAUTH2_CLIENT_SECRET must be given.\n")
		fmt.Fprintf(os.Stderr, "access to https://www.pushbullet.com/#settings/clients\n")
		browser.OpenURL("https://www.pushbullet.com/#settings/clients")
		return nil
	}

	redirectURI := fmt.Sprintf("http://localhost:%d/", c.Port)

	pb := pushbullet.New()

	//
	// fetch the authentication code
	//
	authURI := pb.GetAuthURI(pushbulletOAuth2ClientID, redirectURI)
	if err := browser.OpenURL(authURI); err != nil {
		return fmt.Errorf("failed to open the authURI(%s): %v", authURI, err)
	}

	resultChan := make(chan string)
	go minredir.LaunchMinServer(c.Port, minredir.CodeOAuth2Extractor, resultChan)

	authCode := waitForStringChan(resultChan, time.Duration(c.Timeout)*time.Second)
	if authCode == "" {
		return fmt.Errorf("failed or timed out fetching an authentication code")
	}

	//
	// fetch the refresh token
	//
	accessToken, err := pb.FetchAccessToken(pushbulletOAuth2ClientID, pushbulletOAuth2ClientSecret, authCode)
	if err != nil {
		return fmt.Errorf("failed or timed out fetching the access token: %v", err)
	}

	//
	// store the token to the config file.
	//
	config.Pushbullet.AccessToken = accessToken
	saveConfig(config, global.Config)

	return nil

}

func init() {
	appendCommand(&pbCmd{}, "pushbullet, pb", "")
}

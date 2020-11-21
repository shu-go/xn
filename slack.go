package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/andrew-d/go-termutil"
	"github.com/pkg/browser"
	api "github.com/slack-go/slack"

	"github.com/shu-go/rog"
	"github.com/shu-go/xn/charconv"
	"github.com/shu-go/xn/client/slack"
	"github.com/shu-go/xn/minredir"
)

var (
	SLACK_OAUTH2_CLIENT_ID     string = ""
	SLACK_OAUTH2_CLIENT_SECRET string = ""
)

type slackCmd struct {
	_    struct{}     `help:"notify by slack"`
	Send slackSendCmd `help:"send a notification"`
	Auth slackAuthCmd
}

type slackSendCmd struct {
	Chan   string `default:"general", help:"channel or group name (sub-match, posting to all matching channels and groups, no #)"`
	User   string `help:"user name"`
	Icon   string `help:"message icon"`
	Text   string `help:"message text, or in arguments"`
	Upload string `help:"filename"`
}

type slackAuthCmd struct {
	_       struct{} `help:"authenticate"   usage:"1. go to https://api.slack.com/apps\n2. make a new app\n3. xn slack auth CLIENT_ID CLIENT_SECRET"`
	Port    int      `cli:"port=PORT" default:"7878" help:"a temporal PORT for OAuth authentication."`
	Timeout int      `cli:"timeout=TIMEOUT" default:"60" help:"set TIMEOUT (in seconds) on authentication transaction. < 0 is infinite."`
}

func (c slackSendCmd) Run(global globalCmd, args []string) error {
	config, _ := loadConfig(global.Config)

	if config.Slack.AccessToken == "" {
		return fmt.Errorf("auth first")
	}

	//
	// prepare
	//

	for _, v := range args {
		if len(c.Text) > 0 {
			c.Text += "\n"
		}
		c.Text += v
	}

	if !termutil.Isatty(os.Stdin.Fd()) {
		bytes, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			bytes = []byte{}
		}

		str, _, err := charconv.Convert(bytes)
		if err != nil {
			return fmt.Errorf("failed to convert charset: %v", err)
		}

		if len(c.Text) == 0 {
			c.Text = str
		} else if len(bytes) != 0 {
			c.Text += "\n" + str
		}
	}

	if len(c.Text) == 0 {
		return nil
	}

	sl := api.New(config.Slack.AccessToken)
	if c.Upload != "" {
		upparams := api.FileUploadParameters{
			File:     c.Upload,
			Channels: []string{c.Chan},
			Title:    c.Text,
		}
		_, err := sl.UploadFile(upparams)
		if err != nil {
			return fmt.Errorf("failed to upload file %v: %v", c.Upload, err)
		}
	} else {
		_, _, err := sl.PostMessage("#"+c.Chan,
			api.MsgOptionText(c.Text, true),
			api.MsgOptionUsername(c.User),
			api.MsgOptionIconEmoji(c.Icon))
		if err != nil {
			return fmt.Errorf("failed to post to #%v: %v", c.Chan, err)
		}
	}

	return nil
}

func (c slackAuthCmd) Run(global globalCmd, args []string) error {
	config, _ := loadConfig(global.Config)

	var argClientID, argCLientSecret string
	if len(args) >= 2 {
		argClientID = args[0]
		argCLientSecret = args[1]
	}

	//
	// prepare
	//
	SLACK_OAUTH2_CLIENT_ID = firstNonEmpty(
		argClientID,
		config.Slack.ClientID,
		os.Getenv("SLACK_OAUTH2_CLIENT_ID"),
		SLACK_OAUTH2_CLIENT_ID)
	SLACK_OAUTH2_CLIENT_SECRET = firstNonEmpty(
		argCLientSecret,
		config.Slack.ClientSecret,
		os.Getenv("SLACK_OAUTH2_CLIENT_SECRET"),
		SLACK_OAUTH2_CLIENT_SECRET)

	if SLACK_OAUTH2_CLIENT_ID == "" || SLACK_OAUTH2_CLIENT_SECRET == "" {
		fmt.Fprintf(os.Stderr, "both SLACK_OAUTH2_CLIENT_ID and SLACK_OAUTH2_CLIENT_SECRET must be given.\n")
		fmt.Fprintf(os.Stderr, "access to https://api.slack.com/apps\n")
		browser.OpenURL("https://api.slack.com/apps")
		return nil
	}

	redirectURI := fmt.Sprintf("http://localhost:%d/", c.Port)

	sl := slack.New()

	//
	// fetch the authentication code
	//
	authURI := sl.GetAuthURI(SLACK_OAUTH2_CLIENT_ID, slack.OAUTH2_SCOPE, redirectURI)
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
	// fetch the access token
	//
	accessToken, err := sl.FetchAccessToken(SLACK_OAUTH2_CLIENT_ID, SLACK_OAUTH2_CLIENT_SECRET, authCode, redirectURI)
	if err != nil {
		return fmt.Errorf("failed or timed out fetching the refresh token: %v", err)
	}

	//
	// store the token to the config file.
	//
	config.Slack.AccessToken = accessToken
	saveConfig(config, global.Config)

	return nil
}

func init() {
	rog.Debug("slack init")
	appendCommand(&slackCmd{}, "slack, sl", "")
}

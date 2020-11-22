package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/andrew-d/go-termutil"
	"github.com/shu-go/xn/charconv"
)

type teamsCmd struct {
	_ struct{} `help:"notify by Microsoft Teams"`

	Send teamsSendCmd
	Auth teamsAuthCmd
}

var (
	teamsWebhookURL string = ""
)

type teamsSendCmd struct {
	_ struct{} `help:"send a notification"`

	Text string
}

type teamsAuthCmd struct {
	_ struct{} `help:"authenticate"`

	WebhookURL string `cli:"url=INCOMING_WEBHOOK_URL"  help:"Incoming Webhook URL of your channel"`
}

func (c teamsAuthCmd) Run(global globalCmd, args []string) error {
	config, _ := loadConfig(global.Config)

	var argWebhookURL string
	if len(args) >= 1 {
		argWebhookURL = args[0]
	}

	//
	// prepare
	//
	teamsWebhookURL = firstNonEmpty(
		argWebhookURL,
		config.Teams.WebhookURL,
		os.Getenv("TEAMS_WEBHOOK_URL"),
		teamsWebhookURL)

	if teamsWebhookURL == "" {
		fmt.Fprintf(os.Stderr, "Incoming Webhook URL is required.\n")
		return nil
	}

	config.Teams.WebhookURL = teamsWebhookURL
	saveConfig(config, global.Config)

	return nil
}

func (c teamsSendCmd) Run(global globalCmd, args []string) error {
	config, _ := loadConfig(global.Config)

	if config.Teams.WebhookURL == "" {
		return fmt.Errorf("auth first")
	}

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

	body := &bytes.Buffer{}
	fmt.Fprintf(body, `{"text":%q}`, c.Text)
	_, err := http.Post(config.Teams.WebhookURL, "application/json", body)
	return err
}

func init() {
	appendCommand(&teamsCmd{}, "teams, tm", "")
}

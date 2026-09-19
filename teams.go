package main

import (
	"bytes"
	"fmt"
	"io"

	"net/http"
	"os"
	"strings"

	"github.com/andrew-d/go-termutil"
	"github.com/shu-go/xn/charconv"
)

type teamsCmd struct {
	_ struct{} `help:"notify by Microsoft Teams"`

	Send teamsSendCmd
	Auth teamsAuthCmd
}

type teamsSendCmd struct {
	_ struct{} `help:"send a notification"`

	Text string
}

type teamsAuthCmd struct {
	_ struct{} `help:"authenticate" usage:"xn does not store the Teams webhook URL in the config file; set it via the XN_TEAMS_WEBHOOK_URL environment variable"`
}

func (c teamsAuthCmd) Run(global globalCmd, args []string) error {
	printSetEnvInstead([2]string{"XN_TEAMS_WEBHOOK_URL", "<your webhook url>"})
	return nil
}

func (c teamsSendCmd) Run(global globalCmd, args []string) error {
	config, _ := loadConfig(global.Config)

	webhookURL := firstNonEmpty(
		config.Teams.WebhookURL,
		os.Getenv("XN_TEAMS_WEBHOOK_URL"))

	if webhookURL == "" {
		return fmt.Errorf("auth first")
	}
	if err := requireHTTPS(webhookURL); err != nil {
		return err
	}

	for _, v := range args {
		if len(c.Text) > 0 {
			c.Text += "\n"
		}
		c.Text += v
	}

	if !termutil.Isatty(os.Stdin.Fd()) {
		bytes, err := io.ReadAll(os.Stdin)
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

	c.Text = strings.ReplaceAll(c.Text, "\\n", "\n")

	if len(c.Text) == 0 {
		return nil
	}

	body := &bytes.Buffer{}
	fmt.Fprintf(body, `{
    "type": "message",
    "attachments":[
        {
            "contentType":"application/vnd.microsoft.card.adaptive",
            "contentUrl":null,
            "content":{
                "$schema":"http://adaptivecards.io/schemas/adaptive-card.json",
                "type":"AdaptiveCard",
                "version":"1.4",
                "body":[
                    { "type": "TextBlock", "wrap": true, "text": %q }
                ]
            }
        }
    ]
}`, c.Text)

	_, err := http.Post(webhookURL, "application/json", body)
	if err != nil {
		return nil
	}

	/*
		buf, _ := ioutil.ReadAll(response.Body)
		println(string(buf))
		response.Body.Close()
	*/

	return err
}

func init() {
	appendCommand(&teamsCmd{}, "teams, tm", "")
}

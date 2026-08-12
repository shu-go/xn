package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/andrew-d/go-termutil"

	"github.com/shu-go/xn/charconv"
)

type discordCmd struct {
	_ struct{} `help:"notify by Discord"`

	Send discordSendCmd `help:"send a notification"`
	Auth discordAuthCmd
}

type discordSendCmd struct {
	_ struct{} `usage:"message content is the joined positional arguments (and stdin). username is empty by default (Discord shows the webhook's configured name)"`

	User string `help:"user name"`
	Text string `help:"message text, or in arguments"`
}

type discordAuthCmd struct {
	_ struct{} `help:"authenticate" usage:"1. in Discord, go to Server Settings > Integrations > Webhooks\n2. create a new webhook (or use an existing one) and copy its Webhook URL\n3. xn discord auth WEBHOOK_URL"`
}

func (c discordAuthCmd) Run(global globalCmd, args []string) error {
	config, _ := loadConfig(global.Config)

	var argWebhookURL string
	if len(args) >= 1 {
		argWebhookURL = args[0]
	}

	webhookURL := firstNonEmpty(
		argWebhookURL,
		config.Discord.WebhookURL,
		os.Getenv("DISCORD_WEBHOOK_URL"))

	if webhookURL == "" {
		fmt.Fprintf(os.Stderr, "Webhook URL is required.\n")
		fmt.Fprintf(os.Stderr, "go to Server Settings > Integrations > Webhooks in Discord and create/copy one.\n")
		return nil
	}

	config.Discord.WebhookURL = webhookURL
	return saveConfig(config, global.Config)
}

func (c discordSendCmd) Run(global globalCmd, args []string) error {
	config, _ := loadConfig(global.Config)

	if config.Discord.WebhookURL == "" {
		return fmt.Errorf("auth first")
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

	if len(c.Text) == 0 {
		return nil
	}

	type discordWebhookPayload struct {
		Content  string `json:"content"`
		Username string `json:"username,omitempty"`
	}
	body, err := json.Marshal(discordWebhookPayload{
		Content:  c.Text,
		Username: c.User,
	})
	if err != nil {
		return err
	}

	resp, err := http.Post(config.Discord.WebhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("discord webhook returned status %s", resp.Status)
	}

	return nil
}

func init() {
	appendCommand(&discordCmd{}, "discord, dc", "")
}

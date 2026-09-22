package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

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

	User   string   `help:"user name"`
	Text   string   `help:"message text, or in arguments"`
	Upload []string `help:"filenames to upload as attachments (comma-separated or repeatable)"`
	Retry  int      `cli:"retry=N" default:"0" help:"retry sending N times on failure (0 = no retry)"`
}

type discordAuthCmd struct {
	_ struct{} `help:"authenticate" usage:"1. in Discord, go to Server Settings > Integrations > Webhooks\n2. create a new webhook (or use an existing one) and copy its Webhook URL\n3. set it via the XN_DISCORD_WEBHOOK_URL environment variable"`
}

func (c discordAuthCmd) Run(global globalCmd, args []string) error {
	printSetEnvInstead([2]string{"XN_DISCORD_WEBHOOK_URL", "<your webhook url>"})
	return nil
}

func (c discordSendCmd) Run(global globalCmd, args []string) error {
	config, _ := loadConfig(global.Config)

	webhookURL := firstNonEmpty(
		config.Discord.WebhookURL,
		os.Getenv("XN_DISCORD_WEBHOOK_URL"))

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

	if len(c.Text) == 0 && len(c.Upload) == 0 {
		return nil
	}

	type discordWebhookPayload struct {
		Content  string `json:"content"`
		Username string `json:"username,omitempty"`
	}
	payload := discordWebhookPayload{
		Content:  c.Text,
		Username: c.User,
	}

	var contentType string
	var bodyBytes []byte

	if len(c.Upload) > 0 {
		buf := &bytes.Buffer{}
		w := multipart.NewWriter(buf)

		payloadJSON, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		if err := w.WriteField("payload_json", string(payloadJSON)); err != nil {
			return err
		}

		for i, upload := range c.Upload {
			f, err := os.Open(upload)
			if err != nil {
				return fmt.Errorf("failed to open %s: %w", upload, err)
			}

			part, err := w.CreateFormFile(fmt.Sprintf("files[%d]", i), filepath.Base(upload))
			if err != nil {
				f.Close()
				return err
			}
			if _, err := io.Copy(part, f); err != nil {
				f.Close()
				return fmt.Errorf("failed to upload %s: %w", upload, err)
			}
			f.Close()
		}

		if err := w.Close(); err != nil {
			return err
		}

		contentType = w.FormDataContentType()
		bodyBytes = buf.Bytes()
	} else {
		jsonBody, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		contentType = "application/json"
		bodyBytes = jsonBody
	}

	return withRetry(c.Retry, func() error {
		resp, err := http.Post(webhookURL, contentType, bytes.NewReader(bodyBytes))
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 300 {
			return fmt.Errorf("discord webhook returned status %s", resp.Status)
		}

		return nil
	})
}

func init() {
	appendCommand(&discordCmd{}, "discord, dc", "")
}

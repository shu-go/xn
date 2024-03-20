package main

//go:generate go install github.com/shu-go/nmfmt/cmd/nmfmtfmt@latest
//go:generate nmfmtfmt gmail.go
import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"os"
	"time"

	"github.com/andrew-d/go-termutil"
	"github.com/pkg/browser"
	"golang.org/x/oauth2"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"

	"github.com/shu-go/minredir"
	"github.com/shu-go/nmfmt"
)

var (
	gmailOAuth2ClientID     string = ""
	gmailOAuth2ClientSecret string = ""
)

type gmailCmd struct {
	_    struct{}     `help:"notify by gmail"`
	Send gmailSendCmd `help:"send a notification"`
	Auth gmailAuthCmd
}

type gmailSendCmd struct {
	Subject string `help:"SUBJECT"`
	From    string `help:"FROM address (empty means the authenticated user)"`
	To      string `help:"TO addresses(comma-separated)"`
	CC      string `help:"CC addresses(comma-separated)"`
	BCC     string `help:"BCC addresses(comma-separated)"`
	Body    string `help:"BODY"`

	Timeout int `cli:"timeout=TIMEOUT" default:"60" help:"set TIMEOUT (in seconds) sending a message. < 0 is infinite."`
}

type gmailAuthCmd struct {
	_       struct{} `help:"authenticate (CAUTION: CLIENT_ID and CLIENT_SECRET are stored into a local conf file)"  usage:"1. go to https://console.cloud.google.com\n2. make a new project\n3. go to https://console.cloud.google.com/apis/credentials/consent\n4. finish the consent setting up (name and mail address)\n5. go to https://console.cloud.google.com/apis/dashboard\n6. enable Gmail API\n7. go to https://console.cloud.google.com/apis/credentials\n8. make an OAuth2 Client(Desktop)\n9. xn gmail auth CLIENT_ID CLIENT_SECRET\nCAUTION: CLIENT_ID and CLIENT_SECRET are stored into a local conf file"`
	Port    int      `cli:"port=PORT" default:"7878" help:"a temporal PORT for OAuth authentication."`
	Timeout int      `cli:"timeout=TIMEOUT" default:"60" help:"set TIMEOUT (in seconds) on authentication transaction. < 0 is infinite."`
}

func gmailAuthConfig(clientID, clientSecret string, port int) oauth2.Config {
	redirectURL := fmt.Sprintf("https://localhost:%d/", port)

	return oauth2.Config{
		ClientID:     gmailOAuth2ClientID,
		ClientSecret: gmailOAuth2ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:   "https://accounts.google.com/o/oauth2/auth",
			TokenURL:  "https://oauth2.googleapis.com/token",
			AuthStyle: oauth2.AuthStyleInParams,
		},
		RedirectURL: redirectURL,
		Scopes:      []string{gmail.GmailSendScope},
	}

}

func (c gmailSendCmd) Run(global globalCmd, args []string) error {
	config, _ := loadConfig(global.Config)

	//
	// prepare
	//

	gmailOAuth2ClientID = firstNonEmpty(
		config.Gmail.ClientID,
		os.Getenv("GMAIL_OAUTH2_CLIENT_ID"),
		gmailOAuth2ClientID)
	gmailOAuth2ClientSecret = firstNonEmpty(
		config.Gmail.ClientSecret,
		os.Getenv("GMAIL_OAUTH2_CLIENT_SECRET"),
		gmailOAuth2ClientSecret)

	if config.Gmail.Token == "" || gmailOAuth2ClientID == "" || gmailOAuth2ClientSecret == "" {
		fmt.Fprintf(os.Stderr, "both GMAIL_OAUTH2_CLIENT_ID and GMAIL_OAUTH2_CLIENT_SECRET must be given.\n")
		fmt.Fprintf(os.Stderr, "access to https://console.developers.google.com/apis/credentials\n")
		return nil
	}

	c.From = firstNonEmpty(c.From, "me")
	c.Subject = mime.BEncoding.Encode("UTF-8", c.Subject)

	if !termutil.Isatty(os.Stdin.Fd()) {
		bytes, err := io.ReadAll(os.Stdin)
		if err != nil {
			bytes = []byte{}
		}
		if len(c.Body) == 0 {
			c.Body = string(bytes)
		} else {
			c.Body += "\r\n" + string(bytes)
		}
	}

	for _, b := range args {
		if len(c.Body) != 0 {
			c.Body += "\r\n"
		}
		c.Body += b
	}

	if len(c.Body) == 0 {
		return nil
	}

	var toheader string
	if len(c.To) > 0 {
		toheader = fmt.Sprintf("To: %s\r\n", c.To)
	}
	var ccheader string
	if len(c.CC) > 0 {
		ccheader = fmt.Sprintf("CC: %s\r\n", c.CC)
	}
	var bccheader string
	if len(c.BCC) > 0 {
		bccheader = fmt.Sprintf("BCC: %s\r\n", c.BCC)
	}
	type M map[string]any
	rawMsg := []byte(nmfmt.Sprintf(
		"${To}${CC}${BCC}From: ${From}\r\nSubject: ${Subject}\r\n\r\n${Body}\r\n",
		nmfmt.M{
			"To":      toheader,
			"CC":      ccheader,
			"BCC":     bccheader,
			"From":    c.From,
			"Subject": c.Subject,
			"Body":    c.Body,
		}))

	oauthConfig := gmailAuthConfig(
		gmailOAuth2ClientID,
		gmailOAuth2ClientSecret,
		-1,
	)

	tokBuf := bytes.NewBufferString(config.Gmail.Token)
	tok := &oauth2.Token{}
	err := json.NewDecoder(tokBuf).Decode(tok)
	if err != nil {
		return fmt.Errorf("failed to load token: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.Timeout)*time.Second)
	client := oauthConfig.Client(context.Background(), tok)
	srv, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		cancel()
		return fmt.Errorf("Unable to retrieve Gmail client: %v", err)
	}

	msg := gmail.Message{}
	msg.Raw = base64.StdEncoding.EncodeToString(rawMsg)
	_, err = srv.Users.Messages.Send("me", &msg).Do()
	if err != nil {
		cancel()
		return fmt.Errorf("failed to send mail message: %v", err)
	}

	cancel()

	return nil
}

func (c gmailAuthCmd) Run(global globalCmd, args []string) error {
	config, _ := loadConfig(global.Config)

	var argClientID, argCLientSecret string
	if len(args) >= 2 {
		argClientID = args[0]
		argCLientSecret = args[1]
	}

	//
	// prepare
	//

	gmailOAuth2ClientID = firstNonEmpty(
		argClientID,
		config.Gmail.ClientID,
		os.Getenv("GMAIL_OAUTH2_CLIENT_ID"),
		gmailOAuth2ClientID)
	gmailOAuth2ClientSecret = firstNonEmpty(
		argCLientSecret,
		config.Gmail.ClientSecret,
		os.Getenv("GMAIL_OAUTH2_CLIENT_SECRET"),
		gmailOAuth2ClientSecret)

	if gmailOAuth2ClientID == "" || gmailOAuth2ClientSecret == "" {
		fmt.Fprintf(os.Stderr, "both GMAIL_OAUTH2_CLIENT_ID and GMAIL_OAUTH2_CLIENT_SECRET must be given.\n")
		fmt.Fprintf(os.Stderr, "access to https://console.developers.google.com/apis/credentials\n")
		return browser.OpenURL("https://console.developers.google.com/apis/credentials")
	}

	oauthConfig := gmailAuthConfig(
		gmailOAuth2ClientID,
		gmailOAuth2ClientSecret,
		c.Port,
	)

	//
	// fetch the authentication code
	//
	authURL := oauthConfig.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	if err := browser.OpenURL(authURL); err != nil {
		return fmt.Errorf("failed to open the authURI(%s): %v", authURL, err)
	}

	resultChan := make(chan string)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.Timeout)*time.Second)
	err, errChan := minredir.ServeTLS(ctx, fmt.Sprintf(":%v", c.Port), resultChan)
	if err != nil {
		cancel()
		return err
	}

	authCode := waitForStringChan(resultChan, time.Duration(c.Timeout)*time.Second)
	cancel()

	if authCode == "" {
		select {
		case err = <-errChan:
		default:
			err = nil
		}
		return fmt.Errorf("failed or timed out fetching an authentication code: %w", err)
	}

	tok, err := oauthConfig.Exchange(context.TODO(), authCode)
	if err != nil {
		return fmt.Errorf("Unable to retrieve token from web: %v", err)
	}

	tokBuf := bytes.Buffer{}
	if err := json.NewEncoder(&tokBuf).Encode(tok); err != nil {
		return err
	}
	config.Gmail.Token = tokBuf.String()

	config.Gmail.ClientID = gmailOAuth2ClientID
	config.Gmail.ClientSecret = gmailOAuth2ClientSecret

	return saveConfig(config, global.Config)
}

func init() {
	appendCommand(&gmailCmd{}, "gmail, gm", "")
}

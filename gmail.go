package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mime"
	"net/mail"
	"os"
	"time"

	"github.com/andrew-d/go-termutil"
	"github.com/pkg/browser"
	"golang.org/x/oauth2"
	"google.golang.org/api/gmail/v1"

	"github.com/shu-go/xn/minredir"
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
}

type gmailAuthCmd struct {
	_       struct{} `help:"authenticate (CAUTION: CLIENT_ID and CLIENT_SECRET are stored into a local conf file)"  usage:"1. go to https://console.cloud.google.com\n2. make a new project\n3. go to https://console.cloud.google.com/apis/credentials/consent\n4. finish the consent setting up (name and mail address)\n5. go to https://console.cloud.google.com/apis/dashboard\n6. enable Gmail API\n7. go to https://console.cloud.google.com/apis/credentials\n8. make an OAuth2 Client(Desktop)\n9. xn gmail auth CLIENT_ID CLIENT_SECRET\nCAUTION: CLIENT_ID and CLIENT_SECRET are stored into a local conf file"`
	Port    int      `cli:"port=PORT" default:"7878" help:"a temporal PORT for OAuth authentication."`
	Timeout int      `cli:"timeout=TIMEOUT" default:"60" help:"set TIMEOUT (in seconds) on authentication transaction. < 0 is infinite."`
}

func gmailAuthConfig(clientID, clientSecret string, port int) oauth2.Config {
	redirectURL := fmt.Sprintf("http://localhost:%d/", port)

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
		bytes, err := ioutil.ReadAll(os.Stdin)
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

	var rcpts []string
	{
		if len(c.To) > 0 {
			toaddrs, err := mail.ParseAddressList(c.To)
			if err != nil {
				return fmt.Errorf("failed to parse --to: %v", err)
			}
			for _, a := range toaddrs {
				rcpts = append(rcpts, a.Address)
			}
		}

		if len(c.CC) > 0 {
			ccaddrs, err := mail.ParseAddressList(c.CC)
			if err != nil {
				return fmt.Errorf("failed to parse --cc: %v", err)
			}
			for _, a := range ccaddrs {
				rcpts = append(rcpts, a.Address)
			}
		}

		if len(c.BCC) > 0 {
			bccaddrs, err := mail.ParseAddressList(c.BCC)
			if err != nil {
				return fmt.Errorf("failed to parse --bcc: %v", err)
			}
			for _, a := range bccaddrs {
				rcpts = append(rcpts, a.Address)
			}
		}
	}

	var toheader string
	if len(c.To) > 0 {
		toheader = fmt.Sprintf("To: %s\r\n", c.To)
	}
	var ccheader string
	if len(c.CC) > 0 {
		ccheader = fmt.Sprintf("CC: %s\r\n", c.CC)
	}
	rawMsg := []byte(fmt.Sprintf(
		"%s%sFrom: %s\r\nSubject: %s\r\n\r\n%s\r\n",
		toheader, ccheader, c.From, c.Subject, c.Body))

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

	client := oauthConfig.Client(context.Background(), tok)
	srv, err := gmail.New(client)
	if err != nil {
		return fmt.Errorf("Unable to retrieve Gmail client: %v", err)
	}

	msg := gmail.Message{}
	msg.Raw = base64.StdEncoding.EncodeToString(rawMsg)
	_, err = srv.Users.Messages.Send("me", &msg).Do()
	if err != nil {
		return fmt.Errorf("failed to send mail message: %v", err)
	}

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
		browser.OpenURL("https://console.developers.google.com/apis/credentials")
		return nil
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
	go minredir.LaunchMinServer(c.Port, minredir.CodeOAuth2Extractor, resultChan)

	authCode := waitForStringChan(resultChan, time.Duration(c.Timeout)*time.Second)
	if authCode == "" {
		return fmt.Errorf("failed or timed out fetching an authentication code")
	}

	tok, err := oauthConfig.Exchange(context.TODO(), authCode)
	if err != nil {
		return fmt.Errorf("Unable to retrieve token from web: %v", err)
	}

	tokBuf := bytes.Buffer{}
	json.NewEncoder(&tokBuf).Encode(tok)
	config.Gmail.Token = tokBuf.String()

	config.Gmail.ClientID = gmailOAuth2ClientID
	config.Gmail.ClientSecret = gmailOAuth2ClientSecret

	saveConfig(config, global.Config)

	return nil
}

func init() {
	appendCommand(&gmailCmd{}, "gmail, gm", "")
}

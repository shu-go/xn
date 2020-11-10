package main

import (
	"fmt"
	"io/ioutil"
	"mime"
	"net/mail"
	"net/smtp"
	"os"
	"time"

	"github.com/andrew-d/go-termutil"
	"github.com/pkg/browser"

	"github.com/shu-go/xn/client/gmail"
	"github.com/shu-go/xn/minredir"
)

var (
	GMAIL_OAUTH2_CLIENT_ID     string = ""
	GMAIL_OAUTH2_CLIENT_SECRET string = ""
)

type gmailCmd struct {
	_    struct{}     `help:"notify by gmail"`
	Send gmailSendCmd `help:"send a notification"`
	Auth gmailAuthCmd `help:"authenticate"`
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
	Port    int `cli:"port=PORT" default:"7878" help:"a temporal PORT for OAuth authentication."`
	Timeout int `cli:"timeout=TIMEOUT" default:"60" help:"set TIMEOUT (in seconds) on authentication transaction. < 0 is infinite."`
}

func (c gmailSendCmd) Run(global globalCmd, args []string) error {
	config, _ := loadConfig(global.Config)

	//
	// prepare
	//

	GMAIL_OAUTH2_CLIENT_ID = firstNonEmpty(
		config.Gmail.ClientID,
		os.Getenv("GMAIL_OAUTH2_CLIENT_ID"),
		GMAIL_OAUTH2_CLIENT_ID)
	GMAIL_OAUTH2_CLIENT_SECRET = firstNonEmpty(
		config.Gmail.ClientSecret,
		os.Getenv("GMAIL_OAUTH2_CLIENT_SECRET"),
		GMAIL_OAUTH2_CLIENT_SECRET)

	if GMAIL_OAUTH2_CLIENT_ID == "" || GMAIL_OAUTH2_CLIENT_SECRET == "" {
		fmt.Fprintf(os.Stderr, "both GMAIL_OAUTH2_CLIENT_ID and GMAIL_OAUTH2_CLIENT_SECRET must be given.\n")
		fmt.Fprintf(os.Stderr, "access to https://console.developers.google.com/apis/credentials\n")
		return nil
	}

	c.From = firstNonEmpty(c.From, config.Gmail.User)
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

	gm := gmail.New()
	accessToken, err := gm.FetchAccessToken(GMAIL_OAUTH2_CLIENT_ID, GMAIL_OAUTH2_CLIENT_SECRET, config.Gmail.RefreshToken)
	if err != nil {
		return fmt.Errorf("failed to fetch the access token: %v", err)
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
	msg := []byte(fmt.Sprintf(
		"%s%sFrom: %s\r\nSubject: %s\r\n\r\n%s\r\n",
		toheader, ccheader, c.From, c.Subject, c.Body))

	fmt.Printf("rcpts: %#v\n", rcpts)
	fmt.Printf("msg: %s\n", string(msg))
	err = smtp.SendMail(
		"smtp.gmail.com:587",
		gmail.XOAuth2Auth(config.Gmail.User, accessToken),
		c.From,
		rcpts,
		msg,
	)
	if err != nil {
		return fmt.Errorf("failed to send mail message: %v", err)
	}

	return nil
}

func (c gmailAuthCmd) Run(global globalCmd) error {
	config, _ := loadConfig(global.Config)

	//
	// prepare
	//

	GMAIL_OAUTH2_CLIENT_ID = firstNonEmpty(
		config.Gmail.ClientID,
		os.Getenv("GMAIL_OAUTH2_CLIENT_ID"),
		GMAIL_OAUTH2_CLIENT_ID)
	GMAIL_OAUTH2_CLIENT_SECRET = firstNonEmpty(
		config.Gmail.ClientSecret,
		os.Getenv("GMAIL_OAUTH2_CLIENT_SECRET"),
		GMAIL_OAUTH2_CLIENT_SECRET)

	if GMAIL_OAUTH2_CLIENT_ID == "" || GMAIL_OAUTH2_CLIENT_SECRET == "" {
		fmt.Fprintf(os.Stderr, "both GMAIL_OAUTH2_CLIENT_ID and GMAIL_OAUTH2_CLIENT_SECRET must be given.\n")
		fmt.Fprintf(os.Stderr, "access to https://console.developers.google.com/apis/credentials\n")
		browser.OpenURL("https://console.developers.google.com/apis/credentials")
		return nil
	}

	redirectURI := fmt.Sprintf("http://localhost:%d/", c.Port)

	gm := gmail.New()

	//
	// fetch the authentication code
	//
	authURI := gm.GetAuthURI(GMAIL_OAUTH2_CLIENT_ID, gmail.OAUTH2_SCOPE, redirectURI)
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
	refreshToken, err := gm.FetchRefreshToken(GMAIL_OAUTH2_CLIENT_ID, GMAIL_OAUTH2_CLIENT_SECRET, authCode, redirectURI)
	if err != nil {
		return fmt.Errorf("failed or timed out fetching the refresh token: %v", err)
	}

	//
	// store the token to the config file.
	//
	config.Gmail.RefreshToken = refreshToken

	accessToken, err := gm.FetchAccessToken(GMAIL_OAUTH2_CLIENT_ID, GMAIL_OAUTH2_CLIENT_SECRET, refreshToken)
	if err != nil {
		return fmt.Errorf("failed to fetch the access token: %v", err)
	}
	addr, err := gm.EmailAddress(accessToken)
	if err != nil {
		return fmt.Errorf("failed to fetch the email address: %v", err)
	}
	config.Gmail.User = addr

	saveConfig(config, global.Config)

	return nil
}

func init() {
	appendCommand(&gmailCmd{}, "gmail, gm", "")
}

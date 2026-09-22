package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/andrew-d/go-termutil"
	"github.com/pkg/browser"
	"golang.org/x/oauth2"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"

	"github.com/shu-go/minredir"
)

type gmailCmd struct {
	_    struct{}     `help:"notify by gmail"`
	Send gmailSendCmd `help:"send a notification"`
	Auth gmailAuthCmd
}

type gmailSendCmd struct {
	Subject string   `help:"SUBJECT"`
	From    string   `help:"FROM address (empty means the authenticated user)"`
	To      string   `help:"TO addresses(comma-separated)"`
	CC      string   `help:"CC addresses(comma-separated)"`
	BCC     string   `help:"BCC addresses(comma-separated)"`
	Body    string   `help:"BODY"`
	Attach  []string `help:"filenames to attach (comma-separated or repeatable)"`

	Timeout int `cli:"timeout=TIMEOUT" default:"60" help:"set TIMEOUT (in seconds) sending a message. < 0 is infinite."`
	Retry   int `cli:"retry=N" default:"0" help:"retry sending N times on failure (0 = no retry)"`
}

type gmailAuthCmd struct {
	_       struct{} `help:"authenticate (CAUTION: CLIENT_ID and CLIENT_SECRET are stored into a local conf file)"  usage:"1. go to https://console.cloud.google.com\n2. make a new project\n3. go to https://console.cloud.google.com/apis/credentials/consent\n4. finish the consent setting up (name and mail address)\n5. go to https://console.cloud.google.com/apis/dashboard\n6. enable Gmail API\n7. go to https://console.cloud.google.com/apis/credentials\n8. make an OAuth2 Client(Desktop)\n9. xn gmail auth CLIENT_ID CLIENT_SECRET\nCAUTION: CLIENT_ID and CLIENT_SECRET are stored into a local conf file"`
	Port    int      `cli:"port=PORT" default:"7878" help:"a temporal PORT for OAuth authentication."`
	Timeout int      `cli:"timeout=TIMEOUT" default:"60" help:"set TIMEOUT (in seconds) on authentication transaction. < 0 is infinite."`
}

func gmailAuthConfig(clientID, clientSecret string, port int) oauth2.Config {
	redirectURL := fmt.Sprintf("https://localhost:%d/", port)

	return oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
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

	refreshToken := os.Getenv("XN_GMAIL_REFRESH_TOKEN")

	clientID := firstNonEmpty(
		config.Gmail.ClientID,
		os.Getenv("XN_GMAIL_OAUTH2_CLIENT_ID"))
	clientSecret := firstNonEmpty(
		config.Gmail.ClientSecret,
		os.Getenv("XN_GMAIL_OAUTH2_CLIENT_SECRET"))

	if clientID == "" || clientSecret == "" || refreshToken == "" && config.Gmail.Token == "" {
		fmt.Fprintf(os.Stderr, "auth first")
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

	if len(c.Body) == 0 && len(c.Attach) == 0 {
		return nil
	}

	rawMsg, err := gmailBuildRawMessage(c.To, c.CC, c.BCC, c.From, c.Subject, c.Body, c.Attach)
	if err != nil {
		return fmt.Errorf("failed to build message: %w", err)
	}

	var client *http.Client
	oauthConfig := gmailAuthConfig(
		clientID,
		clientSecret,
		-1,
	)

	var tok *oauth2.Token
	if refreshToken != "" {
		tok = &oauth2.Token{RefreshToken: refreshToken}
	} else {
		tok = &oauth2.Token{}
		tokBuf := bytes.NewBufferString(config.Gmail.Token)
		if err = json.NewDecoder(tokBuf).Decode(tok); err != nil {
			return fmt.Errorf("failed to load token: %v", err)
		}
	}

	client = oauthConfig.Client(context.Background(), tok)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.Timeout)*time.Second)
	srv, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		cancel()
		return fmt.Errorf("Unable to retrieve Gmail client: %v", err)
	}

	msg := gmail.Message{}
	msg.Raw = base64.StdEncoding.EncodeToString(rawMsg)
	err = withRetry(c.Retry, func() error {
		_, err := srv.Users.Messages.Send("me", &msg).Do()
		return err
	})
	if err != nil {
		cancel()
		return fmt.Errorf("failed to send mail message: %v", err)
	}

	cancel()

	return nil
}

// gmailBuildRawMessage builds an RFC 5322 message, using multipart/mixed
// (with base64-encoded attachment parts) when attachments are given.
func gmailBuildRawMessage(to, cc, bcc, from, subject, body string, attachments []string) ([]byte, error) {
	headers := []struct{ name, value string }{
		{"To", to},
		{"CC", cc},
		{"BCC", bcc},
		{"From", from},
	}
	for _, h := range headers {
		if strings.ContainsAny(h.value, "\r\n") {
			return nil, fmt.Errorf("%s must not contain CR or LF characters", h.name)
		}
	}

	buf := &bytes.Buffer{}

	if to != "" {
		fmt.Fprintf(buf, "To: %s\r\n", to)
	}
	if cc != "" {
		fmt.Fprintf(buf, "CC: %s\r\n", cc)
	}
	if bcc != "" {
		fmt.Fprintf(buf, "BCC: %s\r\n", bcc)
	}
	fmt.Fprintf(buf, "From: %s\r\n", from)
	fmt.Fprintf(buf, "Subject: %s\r\n", subject)
	buf.WriteString("MIME-Version: 1.0\r\n")

	if len(attachments) == 0 {
		buf.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n\r\n")
		buf.WriteString(body)
		buf.WriteString("\r\n")
		return buf.Bytes(), nil
	}

	mw := multipart.NewWriter(buf)
	fmt.Fprintf(buf, "Content-Type: multipart/mixed; boundary=%q\r\n\r\n", mw.Boundary())

	bodyHeader := textproto.MIMEHeader{}
	bodyHeader.Set("Content-Type", `text/plain; charset="UTF-8"`)
	bodyPart, err := mw.CreatePart(bodyHeader)
	if err != nil {
		return nil, err
	}
	if _, err := bodyPart.Write([]byte(body)); err != nil {
		return nil, err
	}

	for _, path := range attachments {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", path, err)
		}

		ctype := mime.TypeByExtension(filepath.Ext(path))
		if ctype == "" {
			ctype = "application/octet-stream"
		}
		filename := filepath.Base(path)

		ah := textproto.MIMEHeader{}
		ah.Set("Content-Type", fmt.Sprintf("%s; name=%q", ctype, filename))
		ah.Set("Content-Transfer-Encoding", "base64")
		ah.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))

		part, err := mw.CreatePart(ah)
		if err != nil {
			return nil, err
		}

		enc := base64.NewEncoder(base64.StdEncoding, &base64LineWriter{w: part})
		if _, err := enc.Write(data); err != nil {
			return nil, fmt.Errorf("failed to encode %s: %w", path, err)
		}
		if err := enc.Close(); err != nil {
			return nil, err
		}
	}

	if err := mw.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// base64LineWriter wraps base64 output at 76 columns, as conventional for MIME.
type base64LineWriter struct {
	w   io.Writer
	col int
}

func (b *base64LineWriter) Write(p []byte) (int, error) {
	written := len(p)
	for len(p) > 0 {
		remain := min(76-b.col, len(p))
		if _, err := b.w.Write(p[:remain]); err != nil {
			return 0, err
		}
		b.col += remain
		p = p[remain:]
		if b.col == 76 {
			if _, err := b.w.Write([]byte("\r\n")); err != nil {
				return 0, err
			}
			b.col = 0
		}
	}
	return written, nil
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

	credsFromConfigFile := valueFromConfigFile(argClientID, config.Gmail.ClientID) &&
		valueFromConfigFile(argCLientSecret, config.Gmail.ClientSecret)

	clientID := firstNonEmpty(
		argClientID,
		config.Gmail.ClientID,
		os.Getenv("XN_GMAIL_OAUTH2_CLIENT_ID"))
	clientSecret := firstNonEmpty(
		argCLientSecret,
		config.Gmail.ClientSecret,
		os.Getenv("XN_GMAIL_OAUTH2_CLIENT_SECRET"))

	if clientID == "" || clientSecret == "" {
		fmt.Fprintf(os.Stderr, "both GMAIL_OAUTH2_CLIENT_ID and GMAIL_OAUTH2_CLIENT_SECRET must be given.\n")
		fmt.Fprintf(os.Stderr, "access to https://console.developers.google.com/apis/credentials\n")
		return browser.OpenURL("https://console.developers.google.com/apis/credentials")
	}

	oauthConfig := gmailAuthConfig(
		clientID,
		clientSecret,
		c.Port,
	)

	//
	// fetch the authentication code
	//
	state, err := minredir.GenerateState()
	if err != nil {
		return fmt.Errorf("failed to generate state: %w", err)
	}

	authURL := oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
	if err := browser.OpenURL(authURL); err != nil {
		return fmt.Errorf("failed to open the authURI(%s): %v", authURL, err)
	}

	resultChan := make(chan string)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.Timeout)*time.Second)
	errChan, err := minredir.ServeTLS(ctx, fmt.Sprintf(":%v", c.Port), resultChan, minredir.State(state))
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

	// Only persist the newly obtained token to the config file if the client
	// credentials used to obtain it were themselves loaded from the config
	// file. Otherwise the user is deliberately keeping credentials out of it
	// (CLI arg / env var), so hand the token back via stdout instead.
	if !credsFromConfigFile {
		printSetEnvInstead(
			[2]string{"XN_GMAIL_OAUTH2_CLIENT_ID", clientID},
			[2]string{"XN_GMAIL_OAUTH2_CLIENT_SECRET", clientSecret},
			[2]string{"XN_GMAIL_REFRESH_TOKEN", tok.RefreshToken},
		)
		return nil
	}

	tokBuf := bytes.Buffer{}
	if err := json.NewEncoder(&tokBuf).Encode(tok); err != nil {
		return err
	}
	config.Gmail.Token = tokBuf.String()

	config.Gmail.ClientID = clientID
	// Do not write the client secret received via command-line arguments or
	// environment variables to the configuration file.
	//config.Gmail.ClientSecret = clientSecret

	return saveConfig(config, global.Config)
}

func init() {
	appendCommand(&gmailCmd{}, "gmail, gm", "")
}

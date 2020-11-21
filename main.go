package main

import (
	"os"
	"time"

	"github.com/shu-go/gli"
)

// Version is app version
var Version string

func init() {
	if Version == "" {
		Version = "dev-" + time.Now().Format("20060102")
	}
}

type globalCmd struct {
	Config string `cli:"config, conf=CONFIG_FILE" help:"load configurations from CONFIG_FILE (default: ./xn.conf or EXE_PATH/xn.conf)" `
}

var (
	g_app = gli.NewWith(&globalCmd{})
)

func main() {
	g_app.Name = "xn"
	g_app.Desc = "multi-way notification sender tool"
	g_app.Usage = `
# Slack
# auth
xn slack auth
  1. go to https://api.slack.com/apps
  2. make a new app
  3. xn slack auth CLIENT_ID CLIENT_SECRET
# send
xn slack send testtesttest
# about 'send'
xn slack  help  send
    `
	g_app.Version = Version
	g_app.Copyright = "(C) 2020 Shuhei Kubota"
	if err := g_app.Run(os.Args); err != nil {
		os.Exit(1)
	}

	return
}

func appendCommand(ptrSt interface{}, names, help string) {
	g_app.AddExtraCommand(ptrSt, names, help)
}

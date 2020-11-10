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
	Config string `cli:"config, conf=CONFIG_FILE" default:"./xn.conf" help:"load configurations from CONFIG_FILE"`
}

var (
	g_app = gli.NewWith(&globalCmd{})
)

func main() {
	g_app.Name = "xn"
	g_app.Usage = "multi-way notification sender tool"
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

package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/andrew-d/go-termutil"
	api "github.com/mattn/go-gntp"
)

type growlCmd struct {
	_    struct{}     `help:"notify by Growl(GNTP)"`
	Send growlSendCmd `help:"send a notification"`
}

type growlSendCmd struct {
	Server  string `default:"localhost" help:"Growl server"`
	Port    int    `default:"23053" help:"Growl server port"`
	Title   string `default:"xn" help:"title"`
	Event   string `default:"default" help:"event"`
	Message string `help:"message"`
}

func (c growlSendCmd) Run(global globalCmd, args []string) error {
	//
	// prepare
	//

	for _, v := range args {
		if len(c.Message) > 0 {
			c.Message += "\n"
		}
		c.Message += v
	}

	if !termutil.Isatty(os.Stdin.Fd()) {
		bytes, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			bytes = []byte{}
		}
		if len(c.Message) == 0 {
			c.Message = string(bytes)
		} else if len(bytes) != 0 {
			c.Message += "\n" + string(bytes)
		}
	}

	if len(c.Message) == 0 {
		return nil
	}

	//
	// send
	//
	client := api.NewClient()
	client.AppName = "xn"
	client.Server = fmt.Sprintf("%s:%d", c.Server, c.Port)
	n := []api.Notification{{c.Event, c.Event, true}}
	client.Register(n)
	client.Notify(&api.Message{
		Event: c.Event,
		Title: c.Title,
		Text:  c.Message,
		//Icon:        icon,
		//Callback:    url,
		DisplayName: c.Event,
		//Sticky:      *sticky,
	})

	return nil
}

func init() {
	appendCommand(&growlCmd{}, "growl, gr", "")
}

multi-way notification sender tool

[![Go Report Card](https://goreportcard.com/badge/github.com/shu-go/xn)](https://goreportcard.com/report/github.com/shu-go/xn)
![MIT License](https://img.shields.io/badge/License-MIT-blue)

# Usage

## subcommands

```
xn

Sub commands:
  gmail, gm       notify by gmail
  growl, gr       notify by Growl(GNTP)
  pushbullet, pb  notify by Pushbullet
  slack, sl       notify by slack

Options:
  --conf, --config CONFIG_FILE  load configurations from CONFIG_FILE (default: ./xn.conf or EXE_PATH/xn.conf)

Usage:
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

Help sub commands:
  help     xn help subcommnad subsubcommand
  version  show version

(C) 2020 Shuhei Kubota
```

## slack

```
command slack - notify by slack

Sub commands:
  send  send a notification
  auth  authenticate

Global Options:
  --conf, --config CONFIG_FILE  load configurations from CONFIG_FILE (default: ./notiphi.conf)
```

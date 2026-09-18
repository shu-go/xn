multi-way notification sender tool

![MIT License](https://img.shields.io/badge/License-MIT-blue)

# Usage

## subcommands

```
xn

Sub commands:
  discord, dc     notify by Discord
  gmail, gm       notify by gmail
  pushbullet, pb  notify by Pushbullet
  slack, sl       notify by slack
  teams, tm       notify by Microsoft Teams

Options:
  --conf, --config CONFIG_FILE  load configurations from CONFIG_FILE (default: ./xn.conf or EXE_PATH/xn.conf)

Usage:
  # Slack
  # auth
  xn slack auth
    1. go to https://api.slack.com/apps
    2. make a new app
       - note that in "OAuth & Permissions", add https://localhost:7878/ to Redirected URLs and push Save URLs.
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

# env

- `XN_DISCORD_WEBHOOK_URL`
- `XN_GMAIL_OAUTH2_CLIENT_ID`
- `XN_GMAIL_OAUTH2_CLIENT_SECRET`: DEPRECATED. Pass the secret via commandline arguments only with the `auth` subcommand.
- `XN_GMAIL_REFRESH_TOKEN`
- `XN_PUSHBULLET_ACCESS_TOKEN`
- `XN_PUSHBULLET_OAUTH2_CLIENT_ID`
- `XN_PUSHBULLET_OAUTH2_CLIENT_SECRET`: DEPRECATED. Pass the secret via commandline arguments only with the `auth` subcommand.
- `XN_SLACK_ACCESS_TOKEN`
- `XN_SLACK_OAUTH2_CLIENT_ID`
- `XN_SLACK_OAUTH2_CLIENT_SECRET`: DEPRECATED. Pass the secret via commandline arguments only with the `auth` subcommand.
- `XN_TEAMS_WEBHOOK_URL`

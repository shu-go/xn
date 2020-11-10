

# Usage

## subcommands

```
xn

Sub commands:
  growl, gr       notify by Growl(GNTP)
  pushbullet, pb  notify by Pushbullet
  slack, sl       notify by slack

Options:
  --conf, --config CONFIG_FILE  load configurations from CONFIG_FILE (default: ./xn.conf)

Usage:
  # Slack
  # auth
  xn slack auth
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

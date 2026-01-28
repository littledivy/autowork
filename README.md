<img width="600" alt="image" src="https://github.com/user-attachments/assets/9a425cff-b0b6-477c-bf28-792147359735" />

autowork monitors Slack channels for actionable dev tasks and spawns Claude Code sessions to work on them.

### How it works

autowork polls configured Slack channels for new messages and uses Claude to classify if a message describes actionable work. If actionable, creates a git branch and spawns a Claude Code session. Sessions can be resumed later with `autowork open`

For obvious safety reasons, do it for channels that you control or you will expose be all your work chats to Claude. I have set it up to my own private DM only. Your AI slop is your responsibility. **Use at your own risk**.

### Build

Easiest way to get started is via Nix:

```sh
nix run github:littledivy/autowork#main
```

or build from source using `make`.

### Setup

```sh
autowork config
```

You'll need:
- Slack token (`xoxc-...`) - from browser dev tools. Go to Network tab and look for `token=` in requests.
- Slack `d` cookie (`xoxd-...`) - from browser
- Path to your repos directory
- Channel IDs to monitor

This will generate a config at `~/.autowork`

```json
{
  "slack_token": "xoxc-XXX",
  "slack_cookie": "xoxd-XXX",
  "repos_dir": "/Users/divy/gh",
  "poll_interval_seconds": 300,
  "channel_ids": [
    "D0AABBCCZ",
    "C0AABBCCZ"
  ]
}
```

### Usage

```
Usage:
  autowork config    Configure autowork
  autowork check     Check for new messages once
  autowork start     Run as daemon (polls continuously)
  autowork sessions  List pending work sessions
  autowork open <id> Resume a work session
```


## License

Apache 2.0

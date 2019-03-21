# bd-reminder-bot [![Go Report Card](https://goreportcard.com/badge/github.com/nezorflame/bd-reminder-bot)](https://goreportcard.com/report/github.com/nezorflame/bd-reminder-bot) [![Build Status](https://travis-ci.com/nezorflame/bd-reminder-bot.svg?branch=master)](https://travis-ci.com/nezorflame/bd-reminder-bot)

Slack bot to remind your team and team manager about the upcoming birthdays.

Requires Go 1.8+ (1.10+ for Go modules support).

Inspired by `mybot` from RapidLoop at <https://github.com/rapidloop/mybot>

## Install

- with module-aware `go get`:

```bash
go get github.com/nezorflame/bd-reminder-bot
```

- without:

```bash
go get -u github.com/nezorflame/bd-reminder-bot
cd $GOPATH/src/github.com/nezorflame/bd-reminder-bot
go install
```

## Usage

### Flags

| Flag   | Type     | Description                          | Default     |
| ------ | -------- | ------------------------------------ | ----------- |
| config | `string` | Config file name (without extension) | `config`    |
| db     | `string` | BoltDB file location                 | `./bolt.db` |
| debug  | `bool`   | Debug level for logs                 | `false`     |

### Config

Example configuration can be found in `config.example.toml`

### Available commands

| Command  | Description                                                |
| -------- | ---------------------------------------------------------- |
| hi       | Prints the greeting message                                |
| birthday | Prints the amount of days left to the next user's birthday |
| turnoff  | Prints the farewell message and exits (manager only)       |

Before using any command, mention the bot username before the command name, like this:

`@bdreminder hi`

### Limitations

Currently users have to fill their birthday into the `Skype` field (because there is no `Birthday` field available) so that bot could parse it.

This behaviour will change in the future versions.

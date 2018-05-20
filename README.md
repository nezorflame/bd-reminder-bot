# bd-reminder-bot [![Go Report Card](https://goreportcard.com/badge/github.com/nezorflame/bd-reminder-bot)](https://goreportcard.com/report/github.com/nezorflame/bd-reminder-bot) [![Build Status](https://travis-ci.com/nezorflame/bd-reminder-bot.svg?branch=master)](https://travis-ci.com/nezorflame/bd-reminder-bot) [![codecov](https://codecov.io/gh/nezorflame/bd-reminder-bot/branch/master/graph/badge.svg)](https://codecov.io/gh/nezorflame/bd-reminder-bot)

Slack bot to remind your team and team manager about the upcoming birthdays.

Requires Go v1.8+

Inspired by `mybot` from RapidLoop at <https://github.com/rapidloop/mybot>

## Install

1. Get the bot:
    ```bash
    go get -u github.com/nezorflame/bd-reminder-bot
    cd $GOPATH/src/github.com/nezorflame/bd-reminder-bot
    ```
2. Install the dependencies and the bot itself:
    - with `golang/dep`:
    ```bash
    go get -u github.com/golang/dep/cmd/dep
    dep ensure
    go install
    ```
    - with `vgo`:
    ```bash
    go get -u golang.org/x/vgo
    vgo install
    ```

## Usage

Flags:

| Flag | Type | Description | Default |
|--------|--------|-------------------------------------|-----------|
| config | `string` | Config file name (without extension) | `config` |
| db | `string` | BoltDB file location | `./bolt.db` |
| debug | `bool` | Debug level for logs | `false` |

Example configuration can be found in `config.example.toml`

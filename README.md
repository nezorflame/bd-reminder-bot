# bd-reminder-bot

Slack bot to remind your team and team manager about the upcoming birthdays.

Inspired by `mybot` from RapidLoop at <https://github.com/rapidloop/mybot>

## Usage

Flags:

| Flag | Type | Description | Default |
|--------|--------|-------------------------------------|-----------|
| config | `string` | Config file name (without extension) | `config` |
| db | `string` | BoltDB file location | `./bolt.db` |
| debug | `bool` | Debug level for logs | `false` |

Example configuration can be found in `config.example.toml`

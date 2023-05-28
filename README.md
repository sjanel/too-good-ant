# Too good ant

Command line automated application communicating with Too Good to Go API.

Currently only list available stores and send email from a Gmail account once new stores are found.

Note: **This app is currently under development**

## Install

Easy with [go](https://go.dev/dl/):

Compile with `go build .` from current directory.

## Config

Based on [data/example_config.json](data/example_config.json), write your own private configuration in `secrets/config.json`.

The minimum configuration changes that you need to update is obviously the email accounts, the origin (latitude, longitude) of the center of the search and the `sendConfig` information (`sendConfig.sendAction` can be set to `email`, `whatsapp` or an empty string to disable notifications).

You can define several accounts (with emails) in `tooGoodToGoConfig.accountsEmail` so that they can be used as rolling accounts (starting from the first one) in case one gets too many requests error.

### Send email configuration

With Google gmail API. Currently only works with Gmail accounts, documentation to be done.

Only **send email authorization** is asked by the program.

When ant finds new available bags, it will send emails to addresses defined in the configuration file.

### What's App message connector

If you wish to be alerted by What's App, you can set `sendConfig.sendAction` to `whatsapp` and the tool will first ask to register a new device thanks to a QR code authentification.
This step is only required for the first connection - then you credentials will be stored in a secret file (do not publish it anywhere) `secrets/whatsapp.db` for the next runs.

Set either a **group name**  (`sendConfig.whatsAppConfig.groupNameTo`) or a **user name** (`sendConfig.whatsAppConfig.userNameTo`) that will receive this application's messages.

## Usage

Launch with `./too-good-ant` and let the ant harvest for you.

The whole configuration is provided by file `secrets/config.json`, `verbose` mode can be overridden by command line option `-v`.
`-q` (quiet) allows to force disable verbose mode.

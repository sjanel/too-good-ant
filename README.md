# Too good ant

Command line automated application communicating with Too Good to Go API.

Currently only list available stores and send email from a Gmail account once new stores are found.

Note: **This app is currently under development**

## Install

Easy with [go](https://go.dev/dl/):

Compile with `go build .` from current directory.

## Config

Based on [data/example_config.json](data/example_config.json), write your own private configuration in `secrets/config.json`.

### Send email configuration

With Google gmail API. Currently only works with Gmail accounts, documentation to be done.

Only **send email authorization** is asked by the program.

When ant finds new available bags, it will send emails to addresses defined in the configuration file.

## Usage

Launch with `./too-good-ant` and let the ant harvest for you.

The whole configuration is provided by file `secrets/config.json`, `verbose` mode can be overridden by command line option `-v`.
`-q` (quiet) allows to force disable verbose mode.

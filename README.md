# relayscan

[![Goreport status](https://goreportcard.com/badge/github.com/flashbots/relayscan)](https://goreportcard.com/report/github.com/flashbots/relayscan)
[![Test status](https://github.com/flashbots/relayscan/workflows/Checks/badge.svg)](https://github.com/flashbots/relayscan/actions?query=workflow%3A%22Checks%22)
[![Docker hub](https://badgen.net/docker/size/flashbots/relayscan?icon=docker&label=image)](https://hub.docker.com/r/flashbots/relayscan/tags)

Monitoring, analytics & data for Ethereum MEV-Boost builders and relays

**Running on https://relayscan.io**

A set of tools to fill and show a postgres database.
## Notes

- Work in progress
- Multiple relays can serve a payload for the same slot (if the winning builder sent the best bid to multiple relays, and the proposer asks for a payload from all of them)
- Comments and feature requests: [@relayscan_io](https://twitter.com/relayscan_io)
- License: AGPL
- Maintainer: [@metachris](https://twitter.com/metachris)

---

## Overview

* Uses PostgreSQL as data store
* Relays are configured in [`/common/relays.go`](/common/relays.go)
* Some environment variables are required, see [`.env.example`](/.env.example)
* Saving and checking payloads is split into phases/commands:
  * [`data-api-backfill`](https://github.com/flashbots/relayscan/blob/cleanup/cmd/data-api-backfill.go) -- queries the data API of all relays and puts that data into the database
  * [`check-payload-value`](https://github.com/flashbots/relayscan/blob/cleanup/cmd/check-payload-value.go) -- checks all new database entries for payment validity


## Getting started

### Run

```bash
# Query relay data APIs for delivered payloads, and store in the database (by default, until the merge!)
go run . data-api-backfill

# Backfill data only until a specific slot
go run . data-api-backfill --min-slot=6482170

# Check new entries for valid payments
go run . check-payload-value

# Start the website (--dev reloads the template on every page load, for easier iteration)
go run . website --dev

# Start service to query every relay for bids
go run . collect-live-bids

# Update builder inclusion stats
go run . update-builder-stats --start 2023-06-04 --end 2023-06-06 --daily
go run . update-builder-stats --backfill --daily
go run . update-builder-stats --start 2023-06-04 --end 2023-06-06 --daily --hourly
go run . update-builder-stats --start "2023-06-04 10:00" --end "2023-06-06 18:00" --hourly
```

### Test & dev

```bash
# Install dependencies
go install mvdan.cc/gofumpt@latest
go install honnef.co/go/tools/cmd/staticcheck@v0.3.3
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.49.0

# Lint, test and build
make lint
make test
make test-race
make build
```

### Help

```bash
$ go run . --help
https://github.com/flashbots/relayscan

Usage:
  relay [flags]
  relay [command]

Available Commands:
  backfill-extradata  Backfill extra_data
  check-payload-value Check payload value for delivered payloads
  collect-live-bids   On every slot, ask for live bids
  completion          Generate the autocompletion script for the specified shell
  data-api-backfill   Backfill all relays data API
  help                Help about any command
  inspect-block       Inspect a block
  version             Print the version number the relay application
  website             Start the website server

Flags:
  -h, --help   help for relay

Use "relay [command] --help" for more information about a command.
```
# relayscan

[![Test status](https://github.com/flashbots/relayscan/workflows/Checks/badge.svg)](https://github.com/flashbots/relayscan/actions?query=workflow%3A%22Checks%22)

Disclaimer: This code is work-in-progress and quick'n dirty in a lot of places. Use at your own risk.

Running on https://relayscan.io

---

Monitoring of Ethereum mev-boost relays:

* PostgreSQL database with delivered payloads of all relays
* Ensure payments are correct (according to claim in bid)
* Call `getHeader` on every relay on every slot and save to DB

---

* License: AGPL
* Maintainer: [@metachris](https://twitter.com/metachris)

---

## Overview

* Uses PostgreSQL as data store
* Relays are configured in [`/common/relays.go`](/common/relays.go)
* Some environment variables are required, see [`.env.example`](/.env.example)
* Saving and checking payloads is split into phases/commands:
  * `data-api-backfill` -- queries the data API of all relays and puts that data into the database
  * `check-payload-value` -- checks all new database entries for payment validity

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


## Getting started

### Run

```bash
# Query relay data APIs for delivered payloads
go run . data-api-backfill

# Check new entries for valid payments
go run . check-payload-value

# Start the website (--dev reloads the template on every page load, for easier iteration)
go run . website --dev

# Start service to query every relay for bids
go run . collect-live-bids
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

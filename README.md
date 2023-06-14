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
* Configuration:
  * Relays in [`/vars/relays.go`](/vars/relays.go)
  * Builder aliases in [`/vars/builder_aliases.go`](/vars/builder_aliases.go)
  * Version and common env vars in [`/vars/vars.go`](/vars/vars.go)
* Some environment variables are required, see [`.env.example`](/.env.example)
* Saving and checking payloads is split into phases/commands:
  * [`data-api-backfill`](/cmd/core/data-api-backfill.go) -- queries the data API of all relays and puts that data into the database
  * [`check-payload-value`](/cmd/core/check-payload-value.go) -- checks all new database entries for payment validity
  * [`update-builder-stats`](/cmd/core/update-builder-stats.go) -- create daily builder stats and save to database


## Getting started

### Run

You can either get a copy from the repository and build it yourself, or use the Docker image:

```bash
# Build & run
make build
./relayscan help
./relayscan version

# Run with Docker
docker run flashbots/relayscan
docker run flashbots/relayscan /app/relayscan version

---

# Grab delivered payloads from relays data API, and fill up database
./relayscan core data-api-backfill                     #  for all slots since the merge
./relayscan core data-api-backfill --min-slot 6658658  #  since a given slot (good for dev/testing)

# Double-check new entries for valid payments (and other)
./relayscan core check-payload-value

# Update daily builder inclusion stats
./relayscan core update-builder-stats --start 2023-06-04 --end 2023-06-06  # update daily stats for 2023-06-04 and 2023-06-05
./relayscan core update-builder-stats --start 2023-06-04                   # update daily stats for 2023-06-04 until today
./relayscan core update-builder-stats --backfill                           # update daily stats since last entry, until today

# Start the website (--dev reloads the template on every page load, for easier iteration)
./relayscan service website --dev

# Start service to query every relay for bids
./relayscan service website --dev collect-live-bids
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

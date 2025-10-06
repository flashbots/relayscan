# relayscan

[![Goreport status](https://goreportcard.com/badge/github.com/flashbots/relayscan)](https://goreportcard.com/report/github.com/flashbots/relayscan)
[![Test status](https://github.com/flashbots/relayscan/actions/workflows/checks.yml/badge.svg?branch=main)](https://github.com/flashbots/relayscan/actions?query=workflow%3A%22Checks%22)
[![Docker hub](https://badgen.net/docker/size/flashbots/relayscan?icon=docker&label=image)](https://hub.docker.com/r/flashbots/relayscan/tags)

Monitoring, analytics & data for Ethereum MEV-Boost builders and relays

**Running on https://relayscan.io**

Additional URLs:

- Builder profit:
  - Last 24h: https://www.relayscan.io/builder-profit?t=24h
  - Last 7d: https://www.relayscan.io/builder-profit?t=7d
- Stats in markdown:
  - https://www.relayscan.io/overview/md
  - https://www.relayscan.io/builder-profit/md
- Stats in JSON:
  - https://www.relayscan.io/overview/json
  - https://www.relayscan.io/overview/json?t=7d
  - https://www.relayscan.io/builder-profit/json
  - https://www.relayscan.io/builder-profit/json?t=7d
- Daily stats:
  - https://www.relayscan.io/stats/day/2023-06-20
  - https://www.relayscan.io/stats/day/2023-06-20/json

**Bid Archive**

https://bidarchive.relayscan.io

## Notes

- Work in progress
- At it's core, a set of tools to fill and show a postgres database
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

You can either build relayscan from the repository, or use the Docker image:

```bash
# Build & run
make build
./relayscan help
./relayscan version

# Run with Docker
docker run flashbots/relayscan
docker run flashbots/relayscan /app/relayscan version
```

More example commands:

```bash
# Grab delivered payloads from relays data API, and fill up database
./relayscan core data-api-backfill                     #  for all slots since the merge
./relayscan core data-api-backfill --min-slot 9590900  #  since a given slot (good for dev/testing)

# Double-check new entries for valid payments (and other)
./relayscan core check-payload-value

# Update daily builder inclusion stats
./relayscan core update-builder-stats --start 2023-06-04 --end 2023-06-06  # update daily stats for 2023-06-04 and 2023-06-05
./relayscan core update-builder-stats --start 2023-06-04                   # update daily stats for 2023-06-04 until today
./relayscan core update-builder-stats --backfill                           # update daily stats since last entry, until today

# Start the website (--dev reloads the template on every page load, for easier iteration)
./relayscan service website --dev
```

### Test & development

Start by filling the DB with relay data (delivered payloads), and checking it:

```bash
# Copy .env.example to .env.local, update ETH_NODE_URI and source it
source .env.local

# Start Postgres Docker container
make dev-postgres-start

# Query only a single relay, and for the shortest time possible
go run . core data-api-backfill --relay us --min-slot -2000

# Now the DB has data, check it (and update in DB)
go run . core check-payload-value

# Can also check a single slot only:
go run . core check-payload-value --slot <your_slot>

# Run the website
go run . service website --dev

# Simplify working with read-only DB or large amount of data:
 DB_DONT_APPLY_SCHEMA=1 SKIP_7D_STATS=1 go run . service website --dev

# Now you can open http://localhost:9060 in your browser and see the data
open http://localhost:9060

# You can also reset the database:
make dev-postgres-wipe

# See the Makefile for more commands
make help
```

For linting and testing:

```bash
# Install dependencies
go install mvdan.cc/gofumpt@latest
go install honnef.co/go/tools/cmd/staticcheck@v0.4.3
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.51.2

# Lint and test
make lint
make test
make test-race

# Format the code
make fmt
```


### Updating relayscan

Notes for updating relayscan:

- Relay payloads are selected by `inserted_at`. When adding a new relay, you probably want to manually subtract a day from `inserted_at` so they don't show up all for today (`UPDATE mainnet_data_api_payload_delivered SET inserted_at = inserted_at - INTERVAL '1 DAY' WHERE relay='newrelay.xyz';`). See also https://github.com/flashbots/relayscan/issues/28

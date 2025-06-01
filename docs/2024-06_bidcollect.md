# Relayscan Bid Archive 📚

Relayscan.io collects a full, public archive of bids across [relays](../vars/relays.go).

**https://bidarchive.relayscan.io**

---

### Output

For every day, there are two CSV files:
1. All bids
2. Top bids only

### Bids source types

- `0`: [GetHeader polling](https://ethereum.github.io/builder-specs/#/Builder/getHeader)
- `1`: [Data API polling](https://flashbots.github.io/relay-specs/#/Data/getReceivedBids)
- `2`: [Ultrasound top-bid websocket stream](https://github.com/ultrasoundmoney/docs/blob/main/top-bid-websocket.md)

### Collected fields

| Field                    | Description                                                | Source Types |
| ------------------------ | ---------------------------------------------------------- | ------------ |
| `source_type`            | 0: GetHeader, 1: Data API, 2: Ultrasound stream            | all          |
| `received_at_ms`         | When the bid was first received by the relayscan collector | all          |
| `timestamp_ms`           | When the bid was received by the relay                     | 1 + 2        |
| `slot`                   | Slot the bid was submitted for                             | all          |
| `slot_t_ms`              | How late into the slot the bid was received by the relay   | 1 + 2        |
| `value`                  | Bid value in wei                                           | all          |
| `block_hash`             | Block hash                                                 | all          |
| `parent_hash`            | Parent hash                                                | all          |
| `builder_pubkey`         | Builder pubkey                                             | 1 + 2        |
| `block_number`           | Block number                                               | all          |
| `block_fee_recipient`    | Block fee recipient                                        | 2            |
| `relay`                  | Relay name                                                 | all          |
| `proposer_pubkey`        | Proposer pubkey                                            | 1            |
| `proposer_fee_recipient` | Proposer fee recipient                                     | 1            |
| `optimistic_submission`  | Optimistic submission flag                                 | 1            |

### See also

- Live data: https://bidarchive.relayscan.io
- [Pull request #37](https://github.com/flashbots/relayscan/pull/37)
- [Example output](https://gist.github.com/metachris/061c0443afb8b8d07eed477a848fa395)

---

## Notes on data sources

Source types:
- `0`: `GetHeader` polling
- `1`: Data API polling
- `2`: Ultrasound top-bid Websockets stream

Different data sources have different limitations:

- `GetHeader` polling ([code](/services/bidcollect/getheader-poller.go)):
  - The received header only has limited information, with these implications:
    - Optimistic is always `false`
    - No `builder_pubkey`
    - No bid timestamp (need to use receive timestamp)
    - GetHeader bid timestamps are always when the response from polling at t=1s comes back (but not when the bid was received at a relay)
  - Some relays only allow a single `GetHeader` request per slot, so we time it at `t=1s`
- Data API polling ([code](/services/bidcollect/data-api-poller.go):
    - Has all the necessary information
    - Due to rate limits, we only poll at specific times
    - Polling at t-4, t-2, t-0.5, t+0.5, t+2 (see also [`/services/bidcollect/data-api-poller.go`](/services/bidcollect/data-api-poller.go#64-69))
- Ultrasound websocket stream ([code](/services/bidcollect/ultrasound-stream.go):
  - doesn't expose optimistic, thus that field is always `false`

## Other notes

- Bids are deduplicated based on this key:
  - `fmt.Sprintf("%d-%s-%s-%s-%s", bid.Slot, bid.BlockHash, bid.ParentHash, bid.BuilderPubkey, bid.Value)`
  - this means only the first bid for a given key is stored, even if - for instance - other relays also deliver the same bid
- To store the same bid delivered by different relays use the `--with-duplicates` flag. This will change the deduplication key to:
  - `fmt.Sprintf("%d-%s-%s-%s-%s-%s", bid.Slot, bid.BlockHash, bid.ParentHash, bid.Relay, bid.BuilderPubkey, bid.Value)`
  - this is helpful to measure builder to relay latency.
- Bids can be published to Redis (to be consumed by whatever, i.e. a webserver). The channel is called `bidcollect/bids`.
  - Enable publishing to Redis with the `--redis` flag
  - You can start a webserver that publishes the data via a SSE stream with `--webserver`

---

## Running it

By default, the collector will output CSV into `<outdir>/<date>/<filename>.csv`

```bash
# Start data API and ultrasound stream collectors
go run . service bidcollect --data-api --ultrasound-stream --all-relays

# GetHeader needs a beacon node too
go run . service bidcollect --get-header --beacon-uri http://localhost:3500 --all-relays
```

Publish new bids to Redis:

```bash
# Start Redis
docker run --name redis -d -p 6379:6379 redis

# Start the collector with the `--redis <addr>` flag:
go run . service bidcollect --data-api --ultrasound-stream --redis

# Subscribe to the `bidcollect/bids` channel
redis-cli SUBSCRIBE bidcollect/bids
```

SSE stream of bids via the built-in webserver:

```bash
# Start the webserver in another process to subscribe to Redis and publish bids as SSE stream:
go run . service bidcollect --webserver

# Check if it works by subscribing with curl
curl localhost:8080/v1/sse/bids
```

---

## Useful Clickhouse queries

Useful [clickhouse-local](https://clickhouse.com/docs/en/operations/utilities/clickhouse-local) queries:

```bash
# Set the CSV filename, for ease of reuse across queries
$ fn=2024-06-12_top.csv

# Count different source types in file
$ clickhouse local -q "SELECT source_type, COUNT(source_type) FROM '$fn' GROUP BY source_type ORDER BY source_type;"
0       2929
1       21249
2       1057722

# Count optimistic mode
$ clickhouse local -q "SELECT optimistic_submission, COUNT(optimistic_submission) FROM '$fn' WHERE optimistic_submission IS NOT NULL GROUP BY optimistic_submission;"

# Count bids with >1 ETH in value, by builder
$ clickhouse local -q "SELECT builder_pubkey, count(builder_pubkey) as count, quantile(0.5)(value) as p50, quantile(0.75)(value) as p75, quantile(0.9)(value) as p90, max(value) FROM '$fn' WHERE value > 1000000000000000000 AND builder_pubkey != '' GROUP BY builder_pubkey ORDER BY count DESC FORMAT TabSeparatedWithNames;"

# Get bids > 1 ETH for specific builders
$ clickhouse local -q "SELECT count(value), quantile(0.5)(value) as p50, quantile(0.75)(value) as p75, quantile(0.9)(value) as p90, max(value) FROM '$fn' WHERE value > 1000000000000000000 AND builder_pubkey IN ('0x...', '0x...', '0x...') FORMAT TabSeparatedWithNames;"
```

---

## Architecture

![Architecture](./img/bidcollect-overview.png)


---

## TODO

- Dockerization
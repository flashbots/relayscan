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
- `2`: [Top-bid websocket stream (Ultrasound + Aestus)](https://github.com/ultrasoundmoney/docs/blob/main/top-bid-websocket.md)

### Collected fields

| Field                    | Description                                                | Source Types |
| ------------------------ | ---------------------------------------------------------- | ------------ |
| `source_type`            | 0: GetHeader, 1: Data API, 2: Top-Bid WS Stream            | all          |
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
- `2`: Top-bid Websocket stream

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
- Top-bid websocket stream ([code](/services/bidcollect/top-bid-websocket-stream.go):
  - doesn't expose optimistic, thus that field is always `false`

## Other notes

- Bids are deduplicated based on this key:
  - `fmt.Sprintf("%d-%s-%s-%s-%s", bid.Slot, bid.BlockHash, bid.ParentHash, bid.BuilderPubkey, bid.Value)`
  - this means only the first bid for a given key is stored, even if - for instance - other relays also deliver the same bid

---

## Running it

By default, the collector will output CSV into `<outdir>/<date>/<filename>.csv`

```bash
# Start data API and top-bid websocket stream collectors
go run . service bidcollect --data-api --top-bid-ws-stream --all-relays

# GetHeader needs a beacon node too
go run . service bidcollect --get-header --beacon-uri http://localhost:3500 --all-relays
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
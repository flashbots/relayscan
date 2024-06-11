# Bid Collection

Relayscan should collect bids across relays:

1. [getHeader polling](https://ethereum.github.io/builder-specs/#/Builder/getHeader)
2. [Data API polling](https://flashbots.github.io/relay-specs/#/Data/getReceivedBids)
3. [Ultrasound top-bid websocket stream](https://github.com/ultrasoundmoney/docs/blob/main/top-bid-websocket.md)

Output:

1. CSV file archive
2. Websocket/SSE stream (maybe)

See also:

- [Example output](https://gist.github.com/metachris/061c0443afb8b8d07eed477a848fa395)
- PR: https://github.com/flashbots/relayscan/pull/37
- TODO: link CSV files

---

## Notes on data sources

Source types:
- `0`: `getHeader` polling
- `1`: Data API polling
- `2`: Ultrasound top-bid Websockets stream

Different data sources have different limitations:

- `getHeader` polling:
  - Some relays only allow a single `getHeader` request per slot, so we time it at t=1s
  - Header only has limited information with these implications:
    - Optimistic is always `false`
    - Does not include `builder_pubkey`
    - Does not include bid timestamp (need to use receive timestamp)
    - getHeader bid timestamps are always when the response from polling at t=1s comes back (but not when the bid was received at a relay)
- Data API polling:
    - Has all the necessary information
    - Due to rate limits, we only poll at specific times
    - Polling at t-4, t-2, t-0.5, t+0.5, t+2 (see also [`services/bidcollect/data-api-poller.go`](services/bidcollect/data-api-poller.go#64-69))
  - Ultrasound websocket stream
    - doesn't expose optimistic, thus that field is always `false`

## Other notes

- Bids are deduplicated based on this key:
  - `fmt.Sprintf("%d-%s-%s-%s-%s", bid.Slot, bid.BlockHash, bid.ParentHash, bid.BuilderPubkey, bid.Value)`
  - this means only the first bid for a given key is stored, even if - for instance - other relays also deliver the same bid

---

## Running it

By default, the collector will output CSV into `<outdir>/<date>/<filename>.csv`

```bash
# Start data API and ultrasound stream collectors
go run . service bidcollect --data-api --ultrasound-stream --all-relays

# getHeader needs a beacon node too
go run . service bidcollect --get-header --beacon-uri http://localhost:3500 --all-relays
```

---

## Useful Clickhouse queries

Useful [clickhouse-local](https://clickhouse.com/docs/en/operations/utilities/clickhouse-local) queries:

```bash
clickhouse local -q "SELECT source_type, COUNT(source_type) FROM '2024-06-02_top-00.tsv' GROUP BY source_type ORDER BY source_type;"

# Get bids > 1 ETH for specific builders (CSV has 10M rows)
time clickhouse local -q "SELECT count(value), quantile(0.5)(value) as p50, quantile(0.75)(value) as p75, quantile(0.9)(value) as p90, max(value) FROM '2024-06-05_all.csv' WHERE value > 1000000000000000000 AND builder_pubkey IN ('0xa01a00479f1fa442a8ebadb352be69091d07b0c0a733fae9166dae1b83179e326a968717da175c7363cd5a13e8580e8d', '0xa02a0054ea4ba422c88baccfdb1f43b2c805f01d1475335ea6647f69032da847a41c0e23796c6bed39b0ee11ab9772c6', '0xa03a000b0e3d1dc008f6075a1b1af24e6890bd674c26235ce95ac06e86f2bd3ccf4391df461b9e5d3ca654ef6b9e1ceb') FORMAT TabSeparatedWithNames;"
count(value)    p50     p75     p90     max(value)
1842    1789830446982354000     2279820737908906200     4041286254343376400     8216794401676997763

real    0m2.202s
user    0m17.320s
sys     0m0.589s
```

---

## Architecture

![Architecture](./img/bidcollect-overview.png)


---

## TODO

- spotting some weird lines in csv files, might be concurrent writes or not flushing?
  - -> double-check file contents
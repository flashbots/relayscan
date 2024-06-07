# ADR for bid collection

New and cleaned up doc in [2024-06_bidcollect.md](../2024-06_bidcollect.md).


## Goal

Relayscan should collect bids across relays:

1. [Ultrasound top-bid websocket stream](https://github.com/ultrasoundmoney/docs/blob/main/top-bid-websocket.md)
2. getHeader polling
3. Data API polling

It should expose these as:

1. Parquet/CSV files
2. A websocket/SSE stream

### Notes

- Source types:
  - `0`: `getHeader` polling
  - `1`: Data API polling
  - `2`: Ultrasound top-bid Websockets stream
- getHeader polling
  - some relay only allow a single getHeader request per slot, so we time it at t=1s
  - header only has limited information with these implications:
    - optimistic is always `false`
    - does not include `builder_pubkey`
    - does not include bid timestamp (need to use receive timestamp)
  - Ultrasound relay doesn't support repeated getHeader requests
- Ultrasound websocket stream
  - doesn't expose optimistic, thus that field is always false

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

## Status

Mostly working
- PR: https://github.com/flashbots/relayscan/pull/37
- Example output: https://gist.github.com/metachris/061c0443afb8b8d07eed477a848fa395

Run:

```bash
# CSV output (into `csv/<date>/<filename>.csv`)
go run . service bidcollect --data-api --ultrasound-stream

# TSV output (into `data/<date>/<filename>.tsv`)
go run . service bidcollect --out data --out-tsv --data-api --ultrasound-stream
```

### Done

- Ultrasound bid stream
- Data API polling (at t-4, t-2, t-0.5, t+0.5, t+2)
- getHeader polling at t+1
- CSV/TSV Output
  - Writing to hourly CSV files (one file for top bids, and one for all bids)
  - Cache for deduplication
  - Script to combine into single CSV

### Next up (must have)

- Diagram showing the flow of data and the components involved
- Consider methodology of storing "relay"
- Double-check caching methodology (only one bid per unique key, consider also per source type?)
- Double-check that bids are complete and without duplicates

### Could have

Data API polling
- relay-specific rate limits?

Stream Output
- Websockets or SSE subscription

File Output
- Consider Parquet output files (not sure if needed)
- Upload to S3 + R2 (see also mempool dumpster scripts)

getHeader polling
- some already implemented in [collect-live-bids.go](/cmd/service/collect-live-bids.go))
- define query times

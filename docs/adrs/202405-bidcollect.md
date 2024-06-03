# ADR for bid collection

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
  - header only has limited information. need to use receive timestamp, and optimistic is always false
  - Ultrasound relay doesn't support repeated getHeader requests
- Ultrasound websocket stream
  - doesn't expose optimistic, thus that field is always false

Useful [clickhouse-local](https://clickhouse.com/docs/en/operations/utilities/clickhouse-local) queries:

```bash
clickhouse local -q "SELECT source_type, COUNT(source_type) FROM 'top_2024-06-02_18-00.tsv' GROUP BY source_type ORDER BY source_type;"
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

# ADR for bid stream

## Goal

Relayscan should collect bids across relays:

1. Ultrasound top-bid websocket stream (https://github.com/ultrasoundmoney/docs/blob/main/top-bid-websocket.md)
2. getHeader polling
3. data API polling

It should expose these as:

1. A websocket/SSE stream
2. Parquet/CSV files

## Status

Run:

```
# Ultrasound top-bid stream
go run . service bidcollect --out test.csv --ultrasound-stream

# Data API polling
go run . service bidcollect --out test.csv --data-api
```

Done:

- Ultrasound bid stream
- Data API polling
- Output
  - Writing to csv for top and all bids
  - Cache for deduplication

Next up:

- outputs
  - like mempool dumpster, every N seconds print some stats
  - CSV: dynamic + rotating csv files (like mempool dumpster, for daily files/rollover + combination of multiple collectors)
  - stream (websocket or SSE)

- getHeader polling
  - some already implemented in [collect-live-bids.go](/cmd/service/collect-live-bids.go))
  - define query times

- data API polling
  - relay-specific rate limits

- Collect which source the data is coming from
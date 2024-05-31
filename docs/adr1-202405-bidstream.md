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
go run . service bidcollect --out test.csv
```

Done:

- Ultrasound bid stream works
- Writing to single CSV works

Next up:

- dynamic + rotating csv files (like mempool dumpster, for daily files/rollover + combination of multiple collectors)
- getHeader
  - some already implemented in [collect-live-bids.go](/cmd/service/collect-live-bids.go))
  - define query times
- data API queries
  - relay-specific rate limits
# ADR for bid collection

## Goal

Relayscan should collect bids across relays:

1. Ultrasound top-bid websocket stream (https://github.com/ultrasoundmoney/docs/blob/main/top-bid-websocket.md)
2. getHeader polling
3. data API polling

It should expose these as:

1. Parquet/CSV files
2. A websocket/SSE stream

### Notes

[Source types](https://github.com/flashbots/relayscan/blob/bidstream/services/bidcollect/types.go#L13-L18):
- `0`: `getHeader` polling (not yet implemented)
- `1`: Data API polling
- `2`: Ultrasound top-bid Websockets stream

## Status

Run:

```bash
# Collect bids from ultrasound stream + data API, save to directory "tsv/<date>/<filename>.tsv"
go run . service bidcollect --out tsv --data-api --ultrasound-stream
```

Example output: https://gist.github.com/metachris/061c0443afb8b8d07eed477a848fa395

### Done

- Ultrasound bid stream
- Data API polling (at t-4, t-2, t-0.5, t+0.5, t+2)
- CSV Output
  - Writing to hourly CSV files (one file for top bids, and one for all bids)
  - Cache for deduplication
  - Script to combine into single CSV

### Next up (must have)

- Diagram showing the flow of data and the components involved
- Consider methodology of storing "relay"
- Double-check that bids are complete but without duplicates

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

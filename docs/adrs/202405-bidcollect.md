# ADR for bid collection

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

```bash
# Collect bids from ultrasound stream + data API, save to CSV
go run . service bidcollect --out csv  --data-api --ultrasound-stream
```

### Done

- Ultrasound bid stream
- Data API polling
- Data format
- Output
  - Writing to csv for top and all bids
  - Cache for deduplication

### Next up (must have)

- Diagram showing the flow of data and the components involved
- Consider methodology of storing "relay"
- Double-check that bids are complete but without duplicates
- File Output
  - Combine all individual files into a big file
  - Consider gzipped CSV output: https://gist.github.com/mchirico/6147687 (currently, an hour of bids is about 300MB)
  - Consider Parquet output files
  - Upload to S3 + R2 (see also mempool dumpster scripts)

### Could have

**Data API polling**
- consider improvements to timing
- relay-specific rate limits?

**Stream Output**
- Websockets or SSE subscription

**getHeader polling**
- some already implemented in [collect-live-bids.go](/cmd/service/collect-live-bids.go))
- define query times

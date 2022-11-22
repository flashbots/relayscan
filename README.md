# relayscan

[![Test status](https://github.com/metachris/relayscan/workflows/Checks/badge.svg)](https://github.com/metachris/relayscan/actions?query=workflow%3A%22Checks%22)

Monitoring of Eth2 mev-boost relays:

* Build PostgreSQL with all delivered payloads of all relays
* Ensure payments are correct (according to claim in bid), and save payment details in the DB
* Call `getHeader` on every relay on every slot and save to DB

Running on https://relayscan.io

Website uses:

* https://icons.getbootstrap.com/

---

## Getting started

### Install dependencies

```bash
go install mvdan.cc/gofumpt@latest
go install honnef.co/go/tools/cmd/staticcheck@v0.3.3
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.49.0
```

### Test

```bash
make test
make test-race
make lint
```

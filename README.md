# go-template

[![Test status](https://github.com/metachris/relayscan/workflows/Checks/badge.svg)](https://github.com/metachris/relayscan/actions?query=workflow%3A%22Checks%22)

Starting point for new Go projects:

* Entry file [`main.go`](https://github.com/metachris/relayscan/blob/main/main.go)

---

## Getting started

### Install dependencies

```bash
go install mvdan.cc/gofumpt@latest
go install honnef.co/go/tools/cmd/staticcheck@v0.3.1
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.48.0
```

### Test

```bash
make test
make test-race
make lint
```

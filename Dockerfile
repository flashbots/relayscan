# syntax=docker/dockerfile:1
FROM golang:1.20 as builder
ARG VERSION
WORKDIR /build

# Cache for the modules
COPY go.mod ./
COPY go.sum ./
RUN --mount=type=cache,target=/root/.cache/go-build go mod download

# Now adding all the code and start building
ADD . .
RUN --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -X cmd.Version=${VERSION} -X main.Version=${VERSION}" -v -o relayscan main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/relayscan /app/relayscan
COPY --from=builder /build/services/website/templates/ /app/services/website/templates/
COPY --from=builder /build/static/ /app/static/

ENV LISTEN_ADDR=":8080"
EXPOSE 8080
CMD ["/app/relayscan"]

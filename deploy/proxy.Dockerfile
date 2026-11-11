FROM golang:1.27.4-bookworm AS proxy-builder
ENV CGO_ENABLED=0 GOTOOLCHAIN=local
# Build the proxy with the same patched Go standard library as the app.
WORKDIR /src/proxy
COPY deploy/proxy/go.mod deploy/proxy/go.sum ./
RUN go mod download
COPY deploy/proxy/main.go ./
RUN go build -p 2 -trimpath -tags=nobadger,nomysql,nopgx -ldflags='-s -w' -o /caddy .

FROM caddy:2.11.9-alpine
COPY --from=proxy-builder /caddy /usr/bin/caddy

FROM node:24-bookworm-slim AS web
WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --ignore-scripts
COPY frontend/ ./
RUN npm run build

FROM golang:1.27.4-bookworm AS server
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /gilir ./cmd/gilir

FROM server AS test
CMD ["go", "test", "-race", "-count=1", "./..."]

FROM debian:bookworm-slim AS app
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/*
WORKDIR /app
RUN mkdir -p /app/uploads && chown 10001:10001 /app/uploads
COPY --from=server /gilir /usr/local/bin/gilir
COPY --from=web /src/frontend/dist/ /app/web/
ENV WEB_DIR=/app/web ADDR=:8080 UPLOAD_DIR=/app/uploads
USER 10001:10001
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=4s --start-period=15s --retries=3 CMD ["gilir", "healthcheck"]
ENTRYPOINT ["gilir"]
CMD ["serve"]

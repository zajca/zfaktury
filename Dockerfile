# syntax=docker/dockerfile:1.7

# Build the SvelteKit frontend
FROM node:22-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY frontend/ ./
RUN npm run build

# Build the Go server binary (CGO disabled, server build tag)
FROM golang:1.25-alpine AS backend
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Frontend output must live at web/frontend/build for //go:embed all:frontend/build (relative to web/)
RUN rm -rf web/frontend/build
COPY --from=frontend /app/frontend/build/ ./web/frontend/build/

ARG VERSION=docker
ARG COMMIT=unknown
ARG DATE=unknown
ENV CGO_ENABLED=0
RUN go build -tags server -trimpath \
    -ldflags="-s -w \
      -X github.com/zajca/zfaktury/internal/version.Version=${VERSION} \
      -X github.com/zajca/zfaktury/internal/version.Commit=${COMMIT} \
      -X github.com/zajca/zfaktury/internal/version.Date=${DATE}" \
    -o /out/zfaktury-server ./cmd/zfaktury

# Runtime image: small Alpine with ca-certificates + timezone data
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata wget \
 && addgroup -g 1000 zfaktury \
 && adduser -u 1000 -G zfaktury -S -h /data -D zfaktury \
 && mkdir -p /data && chown -R 1000:1000 /data
COPY --from=backend --chown=1000:1000 /out/zfaktury-server /usr/local/bin/zfaktury-server
USER 1000:1000
ENV ZFAKTURY_DATA_DIR=/data \
    TZ=Europe/Prague
WORKDIR /data
VOLUME ["/data"]
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -qO- http://127.0.0.1:8080/health >/dev/null 2>&1 || exit 1
ENTRYPOINT ["/usr/local/bin/zfaktury-server"]
CMD ["serve", "--host", "0.0.0.0", "--port", "8080"]

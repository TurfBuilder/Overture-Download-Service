# Build stage
# Debian-based image: go-duckdb needs CGO and links cleanest against glibc
FROM golang:1.25 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o overture-downloader ./cmd/overture-downloader

# Runtime stage: distroless keeps only glibc/libstdc++/ca-certs — no shell,
# no package manager. Runs as a non-root user.
FROM gcr.io/distroless/cc-debian12:nonroot

# DuckDB downloads its spatial/httpfs extensions here on first use
ENV HOME=/home/nonroot

WORKDIR /app
COPY --from=builder /app/overture-downloader .

ENTRYPOINT ["./overture-downloader"]

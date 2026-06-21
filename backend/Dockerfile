# --- builder: compile static binary ---
FROM golang:1.23-bookworm AS builder
WORKDIR /app
COPY go.mod ./
COPY go.sum* ./
RUN go mod download || true
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/server ./cmd/server

# --- dev: hot reload friendly, has ffmpeg + go toolchain ---
FROM golang:1.23-bookworm AS dev
RUN apt-get update \
 && apt-get install -y --no-install-recommends ffmpeg ca-certificates \
 && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY go.mod ./
COPY go.sum* ./
RUN go mod download || true
COPY . .
EXPOSE 8080
CMD ["go", "run", "./cmd/server"]

# --- runtime: slim image with only ffmpeg + binary ---
FROM debian:bookworm-slim AS runtime
# wget is required by the ECS container HEALTHCHECK (wget -O- /healthz). The slim
# base doesn't ship it, so without this every deployment fails the health check
# and ECS rolls back. (The dev stage uses the full golang image, which already
# bundles wget — that's why local compose worked but prod deploys didn't.)
RUN apt-get update \
 && apt-get install -y --no-install-recommends ffmpeg ca-certificates wget \
 && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=builder /out/server /app/server
EXPOSE 8080
CMD ["/app/server"]

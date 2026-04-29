FROM golang:1.24.2 AS builder
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    git \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /workspace
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o people_counter_manager ./cmd/server/main.go

FROM debian:bookworm-slim AS runner

RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /workspace
COPY --from=builder /workspace/people_counter_manager .
EXPOSE 9000

CMD ["./people_counter_manager"]




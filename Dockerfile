FROM golang:1.25-alpine AS builder

ENV GOCACHE=/root/.cache/go-build

RUN --mount=type=cache,target=/var/cache/apk,sharing=locked \
    --mount=type=cache,target=/var/lib/apk,sharing=locked \
    apk add --no-cache \
    --repository="https://dl-cdn.alpinelinux.org/alpine/edge/main" \
    --repository="https://dl-cdn.alpinelinux.org/alpine/edge/community" \
    "build-base=0.5-r3" \
    "libheif-dev=1.21.2-r1"

WORKDIR /app

RUN --mount=type=cache,target=/go/pkg/mod \
    go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0

COPY go.mod go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

RUN sqlc generate

RUN --mount=type=cache,target="/root/.cache/go-build" \
    CGO_ENABLED=1 go build \
    -ldflags="-s -w" \
    -o govd ./cmd/main.go

FROM alpine:3.22 AS runtime

WORKDIR /app

RUN --mount=type=cache,target=/var/cache/apk,sharing=locked \
    --mount=type=cache,target=/var/lib/apk,sharing=locked \
    apk add --no-cache \
    --repository="https://dl-cdn.alpinelinux.org/alpine/edge/main" \
    --repository="https://dl-cdn.alpinelinux.org/alpine/edge/community" \
    "ffmpeg=8.0.1-r1" \
    "libheif=1.21.2-r1"

COPY --from=builder /app/govd ./govd

ENTRYPOINT ["./govd"]
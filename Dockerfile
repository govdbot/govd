FROM golang:1.25-alpine

RUN --mount=type=cache,target=/var/cache/apk,sharing=locked \
    --mount=type=cache,target=/var/lib/apk,sharing=locked \
    apk update && \
    apk add --no-cache \
        --repository=https://dl-cdn.alpinelinux.org/alpine/edge/main \
        --repository=https://dl-cdn.alpinelinux.org/alpine/edge/community \
        build-base \
        libheif-dev \
        ffmpeg

WORKDIR /app

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0

RUN sqlc generate

RUN --mount=type=cache,target=/go/pkg/mod \
    go build -o govd ./cmd/main.go

ENTRYPOINT ["./govd"]
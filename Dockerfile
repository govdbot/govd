FROM golang:1.25-alpine

# environment variables for build
ENV GOCACHE=/go-cache
ENV GOMODCACHE=/gomod-cache

# install all dependencies
RUN --mount=type=cache,target=/var/cache/apk,sharing=locked \
    --mount=type=cache,target=/var/lib/apk,sharing=locked \
    apk update && \
    apk add --no-cache \
        --repository=https://dl-cdn.alpinelinux.org/alpine/edge/main \
        --repository=https://dl-cdn.alpinelinux.org/alpine/edge/community \
        build-base \
        pkgconf \
        libheif-dev \
        ffmpeg-dev \
        ffmpeg

WORKDIR /app

# copy go.mod and go.sum first for better caching
COPY go.mod go.sum ./

# download go dependencies and install tools
RUN --mount=type=cache,target=/go-cache \
    --mount=type=cache,target=/gomod-cache \
    go mod download && \
    go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0

# copy the rest of the source code
COPY . .

# generate sqlc code
RUN sqlc generate

# build the application
RUN --mount=type=cache,target=/go-cache \
    --mount=type=cache,target=/gomod-cache \
    go build -ldflags="-s -w" -o govd ./cmd/main.go

ENTRYPOINT ["./govd"]
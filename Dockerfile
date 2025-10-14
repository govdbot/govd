FROM golang:alpine AS builder

# build arguments
ARG LIBHEIF_VERSION=1.20.2

# environment variables for build
ENV CGO_CFLAGS="-I/usr/local/include"
ENV CGO_LDFLAGS="-L/usr/local/lib"
ENV PKG_CONFIG_PATH="/usr/local/lib/pkgconfig"
ENV GOCACHE=/go-cache
ENV GOMODCACHE=/gomod-cache

# install build dependencies first
RUN --mount=type=cache,target=/var/cache/apk,sharing=locked \
    --mount=type=cache,target=/var/lib/apk,sharing=locked \
    apk update && \
    apk add --no-cache \
        git \
        pkgconf \
        build-base \
        tar \
        wget \
        xz \
        gcc \
        cmake \
        libjpeg-turbo-dev \
        libpng-dev \
        libde265-dev \
        x265-dev \
        aom-dev \
        dav1d-dev \
        ffmpeg-dev

# prepare directories
RUN mkdir -p \
    /usr/local/bin \
    /usr/local/lib/pkgconfig \
    /usr/local/lib \
    /usr/local/include \
    /bot/downloads \
    /bot/packages

WORKDIR /bot/packages

# download and build libheif from source
RUN --mount=type=cache,target=/bot/downloads/libheif \
    mkdir -p /bot/downloads/libheif && \
    cd /bot/downloads/libheif && \
    if [ ! -f "libheif-${LIBHEIF_VERSION}.tar.gz" ]; then \
        wget -O "libheif-${LIBHEIF_VERSION}.tar.gz" "https://github.com/strukturag/libheif/releases/download/v${LIBHEIF_VERSION}/libheif-${LIBHEIF_VERSION}.tar.gz"; \
    fi && \
    mkdir -p libheif && \
    tar -xzf "libheif-${LIBHEIF_VERSION}.tar.gz" -C libheif --strip-components=1 && \
    cd libheif && \
    mkdir -p build && \
    cd build && \
    cmake -DCMAKE_BUILD_TYPE=Release .. && \
    make -j"$(nproc)" && \
    make install

WORKDIR /bot

# copy go.mod and go.sum first for better caching
COPY go.mod go.sum ./

# download go dependencies - cached between builds
RUN --mount=type=cache,target=/gomod-cache \
    go mod download && \
    go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

# copy the rest of the source code
COPY . .

# generate sqlc code
RUN sqlc generate

# build the application with build cache
RUN --mount=type=cache,target=/go-cache \
    --mount=type=cache,target=/gomod-cache \
    CGO_ENABLED=1 go build -ldflags="-s -w" -o govd ./cmd/main.go

# final stage - create a smaller runtime image
FROM alpine:latest AS runtime

# install only runtime dependencies with apk cache
RUN --mount=type=cache,target=/var/cache/apk,sharing=locked \
    --mount=type=cache,target=/var/lib/apk,sharing=locked \
    apk update && \
    apk add --no-cache \
        libde265 \
        ca-certificates \
        openssl \
        ffmpeg

# copy libraries and binaries from builder stage
COPY --from=builder /usr/local/lib/ /usr/local/lib/
COPY --from=builder /usr/local/include/ /usr/local/include/
COPY --from=builder /usr/local/lib/pkgconfig/libheif.pc /usr/local/lib/pkgconfig/

# copy the built binary from builder stage
COPY --from=builder /bot/govd /app/govd

WORKDIR /app

ENTRYPOINT ["./govd"]
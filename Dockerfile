FROM golang:bookworm AS builder

# build arguments
ARG FFMPEG_VERSION=7.1
ARG LIBHEIF_VERSION=1.19.7

# environment variables for build
ENV CGO_CFLAGS="-I/usr/local/include"
ENV CGO_LDFLAGS="-L/usr/local/lib"
ENV PKG_CONFIG_PATH="/usr/local/lib/pkgconfig"
ENV GOCACHE=/go-cache
ENV GOMODCACHE=/gomod-cache

# install build dependencies first
RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,target=/var/lib/apt,sharing=locked \
    apt-get update && \
    apt-get upgrade -y && \
    apt-get install -y --no-install-recommends \
        bash \
        git \
        pkg-config \
        build-essential \
        tar \
        wget \
        xz-utils \
        gcc \
        cmake \
        libde265-dev

# prepare directories
RUN mkdir -p \
    /usr/local/bin \
    /usr/local/lib/pkgconfig \
    /usr/local/lib \
    /usr/local/include \
    /bot/downloads \
    /bot/packages

WORKDIR /bot/packages

#bBuild and install libheif - only rebuild if version changes
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
    cmake --preset=release .. && \
    make -j"$(nproc)" && \
    make install

# download and install ffmpeg - only rebuild if version changes
RUN --mount=type=cache,target=/bot/downloads/ffmpeg \
    mkdir -p /bot/downloads/ffmpeg && \
    cd /bot/downloads/ffmpeg && \
    ARCH="$(uname -m)" && \
    if [ "$ARCH" = "aarch64" ] || [ "$ARCH" = "arm64" ]; then \
        FFMPEG_BUILD="https://github.com/BtbN/FFmpeg-Builds/releases/download/latest/ffmpeg-n${FFMPEG_VERSION}-latest-linuxarm64-gpl-shared-${FFMPEG_VERSION}.tar.xz"; \
    else \
        FFMPEG_BUILD="https://github.com/BtbN/FFmpeg-Builds/releases/download/latest/ffmpeg-n${FFMPEG_VERSION}-latest-linux64-gpl-shared-${FFMPEG_VERSION}.tar.xz"; \
    fi && \
    if [ ! -f "ffmpeg-${FFMPEG_VERSION}.tar.xz" ]; then \
        wget -O "ffmpeg-${FFMPEG_VERSION}.tar.xz" "${FFMPEG_BUILD}"; \
    fi && \
    mkdir -p ffmpeg && \
    tar -xf "ffmpeg-${FFMPEG_VERSION}.tar.xz" -C ffmpeg --strip-components=1 && \
    cp -rv ffmpeg/bin/* /usr/local/bin/ && \
    cp -rv ffmpeg/lib/* /usr/local/lib/ && \
    cp -rv ffmpeg/include/* /usr/local/include/ && \
    cp -rv ffmpeg/lib/pkgconfig/* /usr/local/lib/pkgconfig/ && \
    ldconfig /usr/local

WORKDIR /bot

# copy go.mod and go.sum first for better caching
COPY go.mod go.sum ./

# download go dependencies - cached between builds
RUN --mount=type=cache,target=/gomod-cache \
    go mod download && \
    go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

# copy source files needed for sqlc
COPY sqlc.yaml ./
COPY internal/database/queries/ ./internal/database/queries/
COPY internal/database/migrations/ ./internal/database/migrations/

# generate sqlc code
RUN sqlc generate

# copy the rest of the source code
COPY . .

# build the application with build cache
RUN --mount=type=cache,target=/go-cache \
    --mount=type=cache,target=/gomod-cache \
    chmod +x build.sh && ./build.sh

# final stage - create a smaller runtime image
FROM debian:bookworm-slim AS runtime

# install only runtime dependencies with apt cache
RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,target=/var/lib/apt,sharing=locked \
    apt-get update && \
    apt-get install -y --no-install-recommends libde265-0 ca-certificates openssl && \
    rm -rf /var/lib/apt/lists/*

# copy libraries and binaries from builder stage
COPY --from=builder /usr/local/lib/ /usr/local/lib/
COPY --from=builder /usr/local/bin/ /usr/local/bin/

# configure dynamic linker to include /usr/local/lib
RUN ldconfig /usr/local

# copy the built binary from builder stage
COPY --from=builder /bot/govd /app/govd

WORKDIR /app

ENTRYPOINT ["./govd"]
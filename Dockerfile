# Build stage
FROM golang:bookworm AS builder

# Set build arguments
ARG FFMPEG_VERSION=7.1
ARG LIBHEIF_VERSION=1.19.7

# Environment variables for build
ENV CGO_CFLAGS="-I/usr/local/include"
ENV CGO_LDFLAGS="-L/usr/local/lib"
ENV PKG_CONFIG_PATH="/usr/local/lib/pkgconfig"

# Install build dependencies first - these rarely change
RUN apt-get update && \
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
        libde265-dev && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Prepare directories
RUN mkdir -p \
    /usr/local/bin \
    /usr/local/lib/pkgconfig \
    /usr/local/lib \
    /usr/local/include \
    /bot/downloads \
    /bot/packages

# Build and install libheif - only rebuilt if version changes
WORKDIR /bot/packages
RUN wget -O libheif.tar.gz "https://github.com/strukturag/libheif/releases/download/v${LIBHEIF_VERSION}/libheif-${LIBHEIF_VERSION}.tar.gz" && \
    mkdir libheif && \
    tar -xzvf libheif.tar.gz -C libheif --strip-components=1 && \
    rm libheif.tar.gz && \
    cd libheif && \
    mkdir build && \
    cd build && \
    cmake --preset=release .. && \
    make -j"$(nproc)" && \
    make install

# Download and install ffmpeg - only rebuilt if version changes
RUN ARCH="$(uname -m)" && \
    if [ "$ARCH" = "aarch64" ] || [ "$ARCH" = "arm64" ]; then \
        FFMPEG_BUILD="https://github.com/BtbN/FFmpeg-Builds/releases/download/latest/ffmpeg-n${FFMPEG_VERSION}-latest-linuxarm64-gpl-shared-${FFMPEG_VERSION}.tar.xz"; \
    else \
        FFMPEG_BUILD="https://github.com/BtbN/FFmpeg-Builds/releases/download/latest/ffmpeg-n${FFMPEG_VERSION}-latest-linux64-gpl-shared-${FFMPEG_VERSION}.tar.xz"; \
    fi && \
    wget -O ffmpeg.tar.xz "${FFMPEG_BUILD}" && \
    mkdir ffmpeg && \
    tar -xf ffmpeg.tar.xz -C ffmpeg --strip-components=1 && \
    rm ffmpeg.tar.xz && \
    cp -rv ffmpeg/bin/* /usr/local/bin/ && \
    cp -rv ffmpeg/lib/* /usr/local/lib/ && \
    cp -rv ffmpeg/include/* /usr/local/include/ && \
    cp -rv ffmpeg/lib/pkgconfig/* /usr/local/lib/pkgconfig/ && \
    ldconfig /usr/local

# Copy application code last (changes most frequently)
WORKDIR /bot
COPY . .
RUN chmod +x build.sh && ./build.sh

# Final stage - create a smaller runtime image
FROM golang:bookworm AS runtime

# Copy only what's needed from the builder stage
COPY --from=builder /usr/local/lib/ /usr/local/lib/
COPY --from=builder /usr/local/bin/ /usr/local/bin/
COPY --from=builder /bot/govd /app/govd

# Install only runtime dependencies
RUN apt-get update && \
    apt-get install -y --no-install-recommends libde265-0 && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/* && \
    ldconfig /usr/local

WORKDIR /app
ENTRYPOINT ["./govd"]

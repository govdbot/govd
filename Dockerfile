FROM golang:bookworm

ARG FFMPEG_VERSION=7.1
ARG LIBHEIF_VERSION=1.19.7

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

WORKDIR /bot

COPY . .

# check if .env exists, if not create it
RUN if [ ! -f .env ]; then \
        echo ".env file is missing. Creating an empty one..."; \
        touch .env; \
    fi

RUN mkdir -p downloads

WORKDIR /bot/packages

# libheif
ENV LIBHEIF_BUILD="https://github.com/strukturag/libheif/releases/download/v${LIBHEIF_VERSION}/libheif-${LIBHEIF_VERSION}.tar.gz"
RUN wget -O libheif.tar.gz ${LIBHEIF_BUILD} && \
    mkdir -p libheif && \
    tar -xzvf libheif.tar.gz -C libheif --strip-components=1 && \
    rm libheif.tar.gz && \
    cd libheif && \
    mkdir build && \
    cd build && \
    cmake --preset=release .. && \
    make && \
    make install

# ffmpeg
RUN mkdir -p \
    /usr/local/bin \
    /usr/local/lib/pkgconfig/ \
    /usr/local/lib/ \
    /usr/local/include

RUN ARCH=$(uname -m) && \
    if [ "$ARCH" = "aarch64" ] || [ "$ARCH" = "arm64" ]; then \
        echo "detected ARM architecture" && \
        export FFMPEG_BUILD="https://github.com/BtbN/FFmpeg-Builds/releases/download/latest/ffmpeg-n${FFMPEG_VERSION}-latest-linuxarm64-gpl-shared-${FFMPEG_VERSION}.tar.xz"; \
    else \
        echo "detected x86_64 architecture" && \
        export FFMPEG_BUILD="https://github.com/BtbN/FFmpeg-Builds/releases/download/latest/ffmpeg-n${FFMPEG_VERSION}-latest-linux64-gpl-shared-${FFMPEG_VERSION}.tar.xz"; \
    fi && \
    wget -O ffmpeg.tar.xz ${FFMPEG_BUILD} && \
    mkdir -p ffmpeg && \
    tar -xf ffmpeg.tar.xz -C ffmpeg --strip-components=1 && \
    rm ffmpeg.tar.xz && \
    cp -rv ffmpeg/bin/* /usr/local/bin/ && \
    cp -rv ffmpeg/lib/* /usr/local/lib/ && \
    cp -rv ffmpeg/include/* /usr/local/include/ && \
    cp -rv ffmpeg/lib/pkgconfig/* /usr/local/lib/pkgconfig/ && \
    ldconfig /usr/local

# env for building
ENV CGO_CFLAGS="-I/usr/local/include"
ENV CGO_LDFLAGS="-L/usr/local/lib"  
ENV PKG_CONFIG_PATH="/usr/local/lib/pkgconfig"

WORKDIR /bot 

#build
RUN chmod +x build.sh

RUN ./build.sh

ENTRYPOINT ["./govd"]
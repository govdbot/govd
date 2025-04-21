FROM golang:bookworm

# Install dependencies
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
    libjpeg-dev \
    libpng-dev \
    libtiff-dev \
    libgif-dev \
    libde265-dev \
    libaom-dev \
    libx264-dev \
    libx265-dev && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/* 

# download, build and install libheif-1.19.7
RUN wget https://github.com/strukturag/libheif/releases/download/v1.19.7/libheif-1.19.7.tar.gz && \
    tar -xzvf libheif-1.19.7.tar.gz && \
    cd libheif-1.19.7 && \
    mkdir build && \
    cd build && \
    cmake .. && \
    make && \
    make install

# create directories for ffmpeg
RUN mkdir -p /usr/local/bin /usr/local/lib/pkgconfig/ /usr/local/lib/ /usr/local/include

# download and extract FFmpeg 7.0
RUN wget -O ffmpeg.tar.xz https://github.com/BtbN/FFmpeg-Builds/releases/download/autobuild-2024-04-30-12-51/ffmpeg-N-115029-g08781ebe1a-linux64-gpl-shared.tar.xz
    
RUN tar -xf ffmpeg.tar.xz

RUN rm ffmpeg.tar.xz && \
    cp -rv ffmpeg-N-115029-g08781ebe1a-linux64-gpl-shared/bin/* /usr/local/bin/ && \
    cp -rv ffmpeg-N-115029-g08781ebe1a-linux64-gpl-shared/lib/* /usr/local/lib/ && \
    cp -rv ffmpeg-N-115029-g08781ebe1a-linux64-gpl-shared/include/* /usr/local/include/ && \
    cp -rv ffmpeg-N-115029-g08781ebe1a-linux64-gpl-shared/lib/pkgconfig/* /usr/local/lib/pkgconfig/ && \
    ldconfig /usr/local

# set env for building
ENV CGO_CFLAGS="-I/usr/local/include"
ENV CGO_LDFLAGS="-L/usr/local/lib"  
ENV PKG_CONFIG_PATH="/usr/local/lib/pkgconfig"
    
WORKDIR /bot

RUN mkdir -p downloads

COPY . .

RUN chmod +x build.sh

RUN ./build.sh

ENTRYPOINT ["./govd"]
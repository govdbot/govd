FROM golang:alpine

RUN apk update && \
    apk upgrade && \
    apk add --no-cache --repository=https://dl-cdn.alpinelinux.org/alpine/edge/community \
    ffmpeg \
    libheif \
    libheif-dev \
    bash \
    git \
    pkgconfig \
    build-base

WORKDIR /bot

RUN mkdir downloads

COPY . .

RUN chmod +x build.sh

RUN ./build.sh

ENTRYPOINT ["./govd"]
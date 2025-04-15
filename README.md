# govd

a telegram bot for downloading media from various platforms

this project was born after the discontinuation of a highly popular bot known as UVD, and draws significant inspiration from [yt-dlp](https://github.com/yt-dlp/yt-dlp)

- official instance: [@govd_bot](https://t.me/govd_bot)
- support group: [govdsupport](https://t.me/govdsupport)

## features

- download media from various platforms
- download videos, photos, and audio
- inline mode support
- group chat support with customizable settings
- media caption support

## dependencies

- ffmpeg >= 6.1.1
- libheif >= 1.19.7
- pkg-config
- mysql db


## botapi

to avoid limits on files, you should host your own telegram botapi. public bot instance is currently running under a botapi fork, [tdlight-telegram-bot-api](https://github.com/tdlight-team/tdlight-telegram-bot-api)

## installation

```bash
git clone https://github.com/govdbot/govd.git
cd govd
# edit .env file with your bot token and database credentials
sh build.sh
```

## Installation with Docker
first build the image using the dockerfile

```bash
docker build -t govd-bot .
```

then edit the .env file and match the DB properties with the MariaDB service environment variables in the docker-compose.yml file and run

```bash
docker compose up -d
```

## cookies

some extractors require cookies for download. to add your cookies, just insert a txt file in cookies folder (netscape format)

## todo

- [ ] add more extractors
- [ ] switch to sonic json parser
- [ ] switch to native libav
- [ ] add tests
- [ ] add dockerfile and compose
- [ ] improve error handling
- [ ] add support for telegram wehbhooks
- [ ] switch to pgsql (?)
- [ ] better API (?)
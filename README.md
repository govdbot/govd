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

## installation

```bash
git clone https://github.com/govdbot/govd.git
cd govd
# edit .env file with your bot token and database credentials
sh build.sh
```

## installation with docker
first build the image using the dockerfile

```bash
docker build -t govd-bot .
```

next, update the .env file to ensure the database properties match the environment variables defined for the MariaDB service in the docker-compose.yml file 
(while the default environment variables defined for the MariaDB service are acceptable, it is recommended to change the `MYSQL_PASSWORD` property in the docker-compose.yaml file for enhanced security and ensure that you also modify the the `DB_PASSWORD` property in the .env file to reflect this change)

the following line in the .env file MUST be set as shown below

```env
DB_HOST=db
```

finally run the compose to start all services

```bash
docker compose up -d
```

## env variables

| variable              | description                                      | default                      |
|-----------------------|--------------------------------------------------|----------------------------------------|
| `DB_HOST`             | database host                                    | `localhost`                            |
| `DB_PORT`             | database port                                    | `3306`                                 |
| `DB_NAME`             | database name                                    | `govd`                                 |
| `DB_USER`             | database user                                    | `govd`                                 |
| `DB_PASSWORD`         | database password                                | `password`                             |
| `BOT_API_URL`*         | telegram bot api url                             | `https://api.telegram.org`             |
| `BOT_TOKEN`           | telegram bot token                               | `12345678:ABC-DEF1234ghIkl-zyx57W2P0s` |
| `CONCURRENT_UPDATES`  | max concurrent updates handled by the bot        | `50`                                   |
| `LOG_DISPATCHER_ERRORS` | log dispatcher errors        | `0`                                    |
| `DOWNLOADS_DIR`       | directory for downloaded files                   | `downloads`                            |
| `HTTP_PROXY`          | http proxy (optional)                            |                                        |
| `HTTPS_PROXY`         | http proxy (optional)                           |                                        |
| `NO_PROXY`            | no proxy domains (optional)                      |                                        |
| `REPO_URL`            | project repository url                           | `https://github.com/govdbot/govd`      |

**note:**
to avoid limits on files, you should host your own telegram botapi. public bot instance is currently running under a botapi fork, [tdlight-telegram-bot-api](https://github.com/tdlight-team/tdlight-telegram-bot-api), but you can use the official botapi client too.

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
- [ ] better docs with multiple README

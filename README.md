# govd
a telegram bot for downloading media from various platforms

this project was born after the discontinuation of a highly popular bot known as uvd, and draws significant inspiration from [yt-dlp](https://github.com/yt-dlp/yt-dlp)

- official instance: [@govd_bot](https://t.me/govd_bot)
- support group: [govdsupport](https://t.me/govdsupport)

---

* [dependencies](#dependencies)
* [installation](#installation)
    * [build](#build)
    * [docker](#docker-recommended)
* [options](#options)
* [authentication](#authentication)
* [proxying](#proxying)
* [todo](#todo)

# dependencies
- ffmpeg >= 7.x **(*)**
- libheif >= 1.19.7
- pkg-config
- mysql or mariadb

**note:** libav shared libraries must be installed on the system in order to build the bot.

# installation
## build
_this method only works on linux and macos, if you want to build the bot on windows, check [docker installation](#docker-recommended) instead._

1. clone the repository
    ```bash
    git clone https://github.com/govdbot/govd.git && cd govd
    ```

2. edit the `.env` file to set the database properties.  
   for enhanced security, it is recommended to change the `DB_PASSWORD` property in the `.env` file.

3. make sure your database is up and running.

4. build and run the bot:

    ```bash
    sh build.sh && ./govd
    ```

## docker (recommended)

1. build the image using the dockerfile:

    ```bash
    docker build -t govd-bot .
    ```

2. update the `.env` file to ensure the database properties match the environment variables defined for the mariadb service in the `docker-compose.yml` file.  
   for enhanced security, it is recommended to change the `MYSQL_PASSWORD` property in `docker-compose.yaml` and ensure `DB_PASSWORD` in `.env` matches it.

    the following line in the `.env` file **must** be set as:

    ```
    DB_HOST=db
    ```

3. run the compose to start all services:

    ```bash
    docker compose up -d
    ```

# options
| variable                      | description                                  | default                               |
|-------------------------------|----------------------------------------------|---------------------------------------|
| DB_HOST                       | database host                                | localhost                             |
| DB_PORT                       | database port                                | 3306                                  |
| DB_NAME                       | database name                                | govd                                  |
| DB_USER                       | database user                                | govd                                  |
| DB_PASSWORD                   | database password                            | password                              |
| BOT_API_URL                   | telegram bot api url                         | https://api.telegram.org              |
| BOT_TOKEN                     | telegram bot token                           | 12345678:ABC-DEF1234ghIkl-zyx57W2P0s  |
| CONCURRENT_UPDATES            | max concurrent updates handled               | 50                                    |
| LOG_DISPATCHER_ERRORS         | log dispatcher errors                        | 0                                     |
| DOWNLOADS_DIR                 | directory for downloaded files               | downloads                             |
| HTTP_PROXY [(?)](#proxying)   | http proxy (optional)                        |                                       |
| HTTPS_PROXY [(?)](#proxying)  | https proxy (optional)                       |                                       |
| NO_PROXY [(?)](#proxying)     | no proxy domains (optional)                  |                                       |
| REPO_URL                      | project repository url                       | https://github.com/govdbot/govd       |
| PROFILER_PORT                 | port for profiler http server (pprof)        | 0 _(disabled)_                        |

you can configure specific extractors options with `ext-cfg.yaml` file. documentation is not available yet, but you can check the source code for more information.

> [!IMPORTANT]  
> to avoid limits on files, you should host your own telegram botapi and set `BOT_API_URL` variable according. public bot instance is currently running under a botapi fork, [tdlight-telegram-bot-api](https://github.com/tdlight-team/tdlight-telegram-bot-api), but you can use the official botapi client too.

# proxying
there are two types of proxying available: http and edge.
- **http proxy**: this is a standard http proxy that can be used to route requests through a proxy server. you can set the `HTTP_PROXY` and `HTTPS_PROXY` environment variables to use this feature. (SOCKS5 is supported too)
- **edge proxy**: this is a custom proxy that is used to route requests through a specific url. currenrly, you can only set this proxy with `ext-cfg.yaml` file. this is useful for routing requests through a specific server or service. however, this feature is not totally implemented yet.

**note:** by settings `NO_PROXY` environment variable, you can specify domains that should not be proxied.

# authentication
some extractors require authentication to access the content. you can easily use cookies for that; simply export cookies from your browser in netscape format and place them in cookies folder (e.g. `cookies/reddit.txt`). you can easily export cookies using _Get cookies.txt LOCALLY_ extension for your browser.

# todo
- [ ] add more extractors
- [ ] switch to native libav
- [ ] add tests
- [ ] improve error handling
- [ ] add support for telegram webhooks
- [ ] switch to pgsql (?)
- [ ] better api (?)
- [ ] better docs with multiple readme

---
<h1 align="center">govd</h1> <br>
<p align="center">
  <a href="https://t.me/govd_bot">
    <img alt="govd" title="govd" src="https://i.imgur.com/Vx8Psjn.png" width="450">
  </a>
</p>

<p align="center">
  extremely lightweight downloader, inside a telegram bot.
</p>

<p align="center">
    <a href="LICENSE"><img src="https://img.shields.io/github/license/govdbot/govd?style=flat-square" alt="license"></a>
    <img src="https://img.shields.io/badge/docker-ready-blue?style=flat-square" alt="docker">
    <a href="https://t.me/govd_bot"><img src="https://img.shields.io/badge/telegram-@govd__bot-2CA5E0?style=flat-square&logo=telegram" alt="telegram"></a>
</p>

## table of contents

- [features](#features)
- [installation](#installation)
- [configuration](#configuration)
- [docs](#docs)

## features
* vast number of extractors supported
* extremely lightweight
    * minimal memory usage (~80MB)
    * minimal disk usage (~150MB)
* easy to deploy with docker
* highly configurable and extensible
* supports self hosted telegram bot api
* supports authentication for extractors
* available in private chats, groups and inline mode
* translation ready (i18n)

## installation

1. clone the repository:

    ```bash
    git clone https://github.com/govdbot/govd.git && cd govd
    ```

2. edit the `.env` file to match your setup.
make sure the database host is set to:

    ```
    DB_HOST=db
    ``` 

3. start all services with docker:

    ```bash
    docker compose up -d
    ```

## configuration
you can configure the bot using environment variables defined in `.env` file.
refer to `.env.example` file for all available options.

## docs
refer to the [docs](docs/) folder for more information about:
* telegram bot api setup
* extractors configuration
* authentication with extractors
* proxies setup

## migrating from v1
if you are migrating from govd v1 to v2, you can use the provided migration tool. refer to [this page](https://github.com/govdbot/migrate) for more information.

## community
* [official bot](https://t.me/govd_bot)
* [support chat](https://t.me/govdsupport)

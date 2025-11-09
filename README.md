# govd
a telegram bot that lets you download media from various platforms.  

[![license](https://img.shields.io/github/license/govdbot/govd?style=flat-square)](LICENSE)  
[![docker](https://img.shields.io/badge/docker-ready-blue?style=flat-square)](#)  
[![telegram](https://img.shields.io/badge/telegram-@govd__bot-2CA5E0?style=flat-square&logo=telegram)](https://t.me/govd_bot)

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
you can refer to `.env.example` file for all available options.

## extractor options
you can configure specific extractors options with `private/config.yaml` file.
refer to [this page](docs/EXTRACTORS.md) for more information.

## telegram bot api
to avoid limits on files, you should host your own telegram botapi.
refer to [this page](docs/BOTAPI.md) for more information.

## authentication
some extractors require cookies to access the content.
refer to [this page](docs/AUTHENTICATION.md) for more information.

## contributing
contributions are welcome under some strict guidelines.
refer to [this page](docs/CONTRIBUTING.md) for more information.

## migrating from v1
if you are migrating from govd v1 to v2, you can use the provided migration tool. refer to [this page](https://github.com/govdbot/migrate) for more information.

## community
* [official bot](https://t.me/govd_bot)
* [support chat](https://t.me/govdsupport)

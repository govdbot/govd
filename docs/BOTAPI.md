# botapi
hosting your own telegram bot api can help you avoid limits on file sizes and improve performance when downloading media through bot.

you can either host your own bot api using the official [telegram bot api source code](https://github.com/tdlib/telegram-bot-api?tab=readme-ov-file#installation) or use other third-party implementations.

## recommended (with docker)
* [aiogram/telegram-bot-api](https://hub.docker.com/r/aiogram/telegram-bot-api): official telegram bot api, maintained by aiogram team.
* [tdlight/tdlightbotapi](https://hub.docker.com/r/aiogram/telegram-bot-api): a high-performance, memory-optimized bot api, maintained by tdlight team.

## setup your docker compose
since govd uses an internal network with docker, you can either:
* add the bot api service to the existing `docker-compose.yaml` file.
* create a shared network and connect both the bot api and govd services to it. here's an example:

create a new shared network, you can name it as you want:
```bash
docker network create govd-shared
```

start your bot api container connected to the shared network. for example, using `tdlight/tdlightbotapi`:
```bash
docker run -d --name bot-api \
  -e TELEGRAM_API_ID=123 \
  -e TELEGRAM_API_HASH=YOUR_API_HASH \
  -p 8081:8081 \
  --network govd-shared \  # <-- crucial!
  tdlight/tdlightbotapi
```

modify the current govd `docker-compose.yaml` file to connect to the shared network:
```yaml
services:
  bot:
    ...
    networks:
        - govd-network
        - govd-shared
...
networks:
  govd-network:
    driver: bridge
  govd-shared:
    external: true
```

update your environment variable in the `.env` file to point to your bot api service:
```
BOT_API_URL=http://bot-api:8081
```
# setup
youtube extractor is currently based on the invidious api.

## why invidious
* avoids ip bans and rate limits from youtube
* removes the need to bypass captchas or bot protections
* allows you to use public or self-hosted invidious instances
* provides a stable, simplified api for accessing video info and streams

## configuration
you must specify the invidious instance in your `private/config.yaml` file as an array:

```yaml
youtube:
  instance:
    - https://your-invidious-instance.example.com
    - https://another-invidious-instance.example.com
    - ...
```

for more details, see [configuration page](../../../docs/EXTRACTORS.md).

## notes
* if the invidious instance is slow, rate-limited, or offline, video extraction may fail
* choose a reliable public instance or run your own for best results
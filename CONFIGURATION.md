# configuration
the `config.yaml` file allows you to set custom options for each extractor. this is useful for advanced configuration of the bot, mostly related to network settings.
> [!NOTE]
> this configuration will override the global configuration. this is useful in case you want to set a global proxy in the `.env` file and then override it for specific extractors in the `config.yaml` file.

# structure
the file uses yaml format. each top-level key is the name of an extractor. under each extractor, you can define options supported by that extractor, for example:
```yaml
instagram:
  edge_proxy_url: https://example.com
  impersonate: true
  disabled: true

reddit:
  http_proxy: https://example.com
  no_proxy: example.com
```

# available options

## extractor options
* `disabled`: whether to disable the extractor. this is useful for extractors that are not working or are not needed. if you want to disable an extractor, you can set it to `true` in the configuration file.
* `instance`: the instance to use for this extractor. this is useful for extractors that support multiple instances (e.g. [invidious for youtube](ext/youtube/SETUP.md)).

## network options
* `http_proxy` | `https_proxy`: the http(s) proxy to use for this extractor. see [proxying](README.md#proxying) for more information.
* `no_proxy`: the domains that should not be proxied for this extractor. 
* `edge_proxy_url`: the url of the edge proxy to use for this extractor. see [edge proxy](EDGEPROXY.md) for more information.
* `impersonate`: whether to impersonate chrome. this is useful for extractors that require specific browsers' fingerprints to work.
# edge proxy
edge proxy is an optional feature that allows routing some extractor requests through a custom proxy endpoint, instead of a classic http(s)/socks5 proxy. this is useful if you want to centralize or control the traffic of certain platforms via your own proxy service, for example to bypass geo-restrictions, add caching, logging, or other customizations.

## configuration
edge proxy is configured via the `private/config.yaml` file.  
you can set the proxy url for each extractor that supports it.  
example:

```yaml
instagram:
  edge_proxy: https://example.com

reddit:
  proxy: https://example.com
```

## response format
the edge proxy must respond with a JSON object in the following format (see [`networking.EdgeProxyResponse`](../internal/networking/models.go)).

```json
{
  "url": "https://example.com/resource",
  "status_code": 200,
  "text": "response body",
  "headers": {
    "Content-Type": "application/json"
  },
  "cookies": [
    "cookie1=value1; Path=/; HttpOnly",
    "cookie2=value2; Path=/"
  ]
}
```

## notes
* edge proxy is for advanced use and not required for most users.
* this feature is experimental and may change in the future.
* you can check full implementation of the edge proxy in the [`networking`](../internal/networking/edgeproxy.go) package.
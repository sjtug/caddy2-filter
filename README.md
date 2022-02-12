# caddy2-filter

[![Project Status: WIP – Initial development is in progress, but there has not yet been a stable, usable release suitable for the public.](https://www.repostatus.org/badges/latest/wip.svg)](https://www.repostatus.org/#wip)
[![GoDoc](http://img.shields.io/badge/godoc-reference-blue.svg)](https://godoc.org/gopkg.in/sjtug/caddy2-filter)
[![Go Report Card](https://goreportcard.com/badge/github.com/sjtug/caddy2-filter)](https://goreportcard.com/report/github.com/sjtug/caddy2-filter)

Replace text in HTTP response based on regex. Similar to [http.filter](https://caddyserver.com/v1/docs/http.filter) in Caddy 1.

## Usage

Only the listed fields are supported.

The replacement supports capturing groups of search_pattern (e.g. `{1}`) and caddy placeholders (e.g. `{http.request.hostport}`)


Caddyfile:

```caddyfile
# Add this block in top-level settings:
{
    order filter after encode
}

filter {
    # Only process URL matching this regex
    path <optional, regexp pattern, default: .*>
    # Don't process response body larger than this size
    max_size <optional, int, default: 2097152>
    search_pattern <regexp pattern>
    replacement <replacement string>
    # Only process content_type matching this regex
    content_type <regexp pattern>
}

# If you are using reverse_proxy, add this to its config to ensure
# reverse_proxy returns uncompressed body:

header_up -Accept-Encoding
```

JSON config (under `apps › http › servers › routes › handle`)

```json
{
    "handler": "filter",
    "max_size": <int>,
    "path": "<regexp>",
    "search_pattern": "<regexp>",
    "replacement: "<string>",
    "content_type": "<regexp>"
}
```

## Alternatives

As of Jun 2021, <https://github.com/caddyserver/replace-response> can achieve similar functionalities. This plugin's design differs from that one in the following aspects:

1. This plugin supports placeholders like {http.host}
2. This plugin allows capping the max size of buffered response
3. This plugin supports only replacing responses with certain content_types

1. replace-response supports `stream` mode, which features in better performance at the cost of possibilities of omitted replacements
2. replace-response is a semi-official plugin maintained by the same author of caddy

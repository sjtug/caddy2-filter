# caddy2-filter

[![Project Status: WIP – Initial development is in progress, but there has not yet been a stable, usable release suitable for the public.](https://www.repostatus.org/badges/latest/wip.svg)](https://www.repostatus.org/#wip)
[![GoDoc](http://img.shields.io/badge/godoc-reference-blue.svg)](https://godoc.org/gopkg.in/sjtug/caddy2-filter)
[![Go Report Card](https://goreportcard.com/badge/github.com/sjtug/caddy2-filter)](https://goreportcard.com/report/github.com/sjtug/caddy2-filter)

Replace text in HTTP response based on regex. Similar to [http.filter](https://caddyserver.com/v1/docs/http.filter) in Caddy 1.

## Usage

Only the listed fields are supported.

The replacement supports capturing groups of search_pattern (e.g. `{1}`) and caddy placeholders (e.g. `{http.request.hostport}`)


Caddyfile:
```
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
```
{
    "handler": "filter",
    "max_size": <int>,
    "path": "<regexp>",
    "search_pattern": "<regexp>",
    "replacement: "<string>",
    "content_type": "<regexp>"
}
```

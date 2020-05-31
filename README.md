# caddy2-filter

[![Project Status: WIP – Initial development is in progress, but there has not yet been a stable, usable release suitable for the public.](https://www.repostatus.org/badges/latest/wip.svg)](https://www.repostatus.org/#wip)
[![GoDoc](http://img.shields.io/badge/godoc-reference-blue.svg)](https://godoc.org/gopkg.in/sjtug/caddy2-filter)
[![Go Report Card](https://goreportcard.com/badge/github.com/sjtug/caddy2-filter)](https://goreportcard.com/report/github.com/sjtug/caddy2-filter)

Replace text in HTTP response based on regex. Similar to [http.filter](https://caddyserver.com/v1/docs/http.filter) in Caddy 1.

## Usage

Only the listed fields are supported.


Caddyfile:
```
filter {
    search_pattern <regexp pattern>
    replacement <replacement string>
    content_type <regexp pattern>
}
```

JSON config (under `apps › http › servers › routes › handle`)
```
{
    "handler": "filter",
    "search_pattern": "<regexp>",
    "replacement: "<string>",
    "content_type": "<regexp>"
}
```

## Limitation

For response body > 2M, this plugin won't handle it to avoid memory exhaustion from buffering.

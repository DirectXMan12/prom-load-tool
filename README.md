Prometheus Load Tool
====================

This tool generates sequences of periodically-refreshing random Prometheus
metrics.  It can be useful for testing load on Prometheus or intermediate
scrape proxies.

Usage

```shell
# generate 1000 different families, with up to 10 series each
load-tool 1000 10
```

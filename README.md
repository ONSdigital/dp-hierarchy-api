dp-hierarchy-api
================

An API used to retrieve hierarchies - using root or child endpoints -
returning children and parents (breadcrumbs) for the requested node.

### Getting started

### Configuration

| Environment variable         | Default                                  | Description
| ---------------------------- |------------------------------------------| -----------
| BIND_ADDR                    | :22600                                   | The host and port to bind to
| HIERARCHY_API_URL            | http://localhost:22600                   | The external address of this API
| GRACEFUL_SHUTDOWN_TIMEOUT    | 5s                                       | The graceful shutdown timeout (Go `time.Duration` format)
| CODE_LIST_URL                | http://localhost:22400                   | The external address of the Code List API
| HEALTHCHECK_INTERVAL         | 30s                                      | The time between doing health checks
| HEALTHCHECK_CRITICAL_TIMEOUT | 90s                                      | The time taken for the health changes from warning state to critical due to subsystem check failures
| ENABLE_URL_REWRITING         | false                                    | Feature flag to enable URL rewriting

#### Graph / Neptune Configuration

| Environment variable    | Default | Description
| ------------------------| ------- | -----------
| GRAPH_DRIVER_TYPE       | ""      | string identifier for the implementation to be used (e.g. 'neptune' or 'mock')
| GRAPH_ADDR              | ""      | address of the database matching the chosen driver type (web socket)
| NEPTUNE_TLS_SKIP_VERIFY | false   | flag to skip TLS certificate verification, should only be true when run locally

:warning: to connect to a remote Neptune environment on MacOSX using Go 1.18 or higher you must set `NEPTUNE_TLS_SKIP_VERIFY` to true. See our [Neptune guide](https://github.com/ONSdigital/dp/blob/main/guides/NEPTUNE.md) for more details.

### Contributing

See [CONTRIBUTING](CONTRIBUTING.md) for details.

### License

Copyright © 2016-2023, Office for National Statistics (https://www.ons.gov.uk)

Released under MIT license, see [LICENSE](LICENSE.md) for details

dp-hierarchy-api
================

An API used to retrieve hierarchies - using root or child endpoints -
returning children and parents for the requested node.

### Getting started

### Configuration

| Environment variable        | Default                                   | Description
| --------------------------- | ----------------------------------------- | -----------
| BIND_ADDR                   | :22600                                    | The host and port to bind to
| HIERARCHY_API_URL           | http://localhost:22600                    | The external address of this API
| HIERARCHY_DATABASE_ADDRESS  | bolt://localhost:7687                     | The address of the hierarchies database
| GRACEFUL_SHUTDOWN_TIMEOUT   | 5s                                        | The graceful shutdown timeout (Go `time.Duration` format)
| CODE_LIST_URL               | http://localhost:22400                    | The external address of the Code List API
| NEO4J_POOL_SIZE             | 30                                        | The size of the neo4j connection pool

### Contributing

See [CONTRIBUTING](CONTRIBUTING.md) for details.

### License

Copyright © 2016-2017, Office for National Statistics (https://www.ons.gov.uk)

Released under MIT license, see [LICENSE](LICENSE.md) for details

## Directories Overview

### [paginated_poll_listener](./internal/paginated_poll_listener)

Abstraction over a paginated poll listener. It makes it easier to separate how we fetch data, or paginate to get new data, from how we process it.
It says in which order we fetch, or how to store last keys, when we get resolution for data, etc.

### [logstore_listener](internal/extensions/listeners/logstore_listener)

Implements a way to periodically fetch data from the Log Store. Uses the paginated_poll_listener abstraction. It says how keying works and how to fetch data. Keys are pagination keys. For example, we could use as key: timestamps, block heights, etc.

See https://docs.kwil.com/docs/extensions/event-listeners for more information on kwil listeners.

### [ingest_resolution](internal/extensions/resolutions/ingest_resolution)

Implements the consensus mechanisms. It is used by `pagination_poll_listener` to resolve data. For example, it says how to serialize data, and what to do when consensus is reached.

See https://docs.kwil.com/docs/extensions/resolutions for more information on kwil resolutions.

## [demo-contract](./examples/demo-contract)

Provides a simple kuneiform file demonstrating how we can use the `logstore_listener` and `ingest_resolution` to fetch data from the Log Store into a kwil contract.

## How to run

There are docker services being prepared to run the project (see [docker-compose.yml](./docker-compose.yml)). However, this is not fully working yet, and the project is not ready to run.
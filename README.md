## Run it

### Pre-requisites

- `streamr-cli` and basic knowledge of [how to use it](https://docs.streamr.network/usage/cli-tool).
  ```shell
   pnpm install -g @streamr/cli-tools@8.5.5
   ```
- `kwil-cli` and a basic knowledge of [how to use it](https://docs.kwil.com/docs/kwil-cli/installation)
- Some tokens in your polygon wallet to deploy a stream

### Steps to test a single node

1. Create the demo stream
    ```bash
   streamr stream create /kwil-demo
    ```

2. Make the stream public
    ```bash
   streamr stream grant-permission /kwil-demo public subscribe
    ```

3. Adjust the configuration for the log store to start tracking the stream [here](./examples/single-node/logstore-node-config.json)
   ```
     "trackedStreams": [
         {
             "id": "<your_address>/kwil-demo",
             "partitions": 1
         }
     ]
   ```

4. Adjust the configuration of the oracle extension [here](./examples/single-node/config.toml)
   ```toml
   stream_id = "<your_address>/kwil-demo"
   ```

5. Start the docker services
   ```bash
   docker compose -f ./docker-compose.yaml up -d
   ```

6. Deploy the demo schema
   ```bash
   kwil-cli database deploy -p=./examples/demo-contract/demo.kf --name=demo --sync
    ```

7. Publish a message to the stream
    ```bash
   streamr stream publish /kwil-demo
   # then type in some messages in JSON format, such as {"hello": "world"}
    ```

8. (After 2 minutes) Call an action to get data from kwil node
    ```bash
    kwil-cli database call -a=get_data -n=demo
   ```

Verify the output of the last command. It should return the messages you published in the stream.

### Testing a network of nodes

To test a network of Kwil + Log Store nodes, there are some differences in the steps related to nodes configuration and to docker services startup.

3. For each `logstore-config.json` inside the `[examples/testnet](./examples/testnet) > nodeX` directory, adjust the configuration for the log store to start tracking the stream.
   ```
     "trackedStreams": [
         {
             "id": "<your_address>/kwil-demo",
             "partitions": 1
         }
     ]
   ```

4. For each `config.toml` inside the `[examples/testnet](./examples/testnet) > nodeX` directory, adjust the configuration of the oracle extension.
   ```toml
    stream_id = "<your_address>/kwil-demo"
    ```
5. Start the docker services
   ```bash
   docker compose -f ./docker-compose-testnet.yaml up -d
   ```

The other steps are the same as the single node test.

## Directories Overview

### [paginated_poll_listener](./internal/paginated_poll_listener)

Abstraction over a paginated poll listener. It makes it easier to separate how we fetch data or paginate to get new data from how we process it.
It says in which order we fetch, how to store the last keys, when we get data resolution, etc.

### [logstore_listener](internal/extensions/listeners/logstore_listener)

Implements a way to periodically fetch data from the Log Store. It uses the paginated_poll_listener abstraction. It explains how keying works and how to fetch data. Keys are pagination keys. For example, we could use timestamps, block heights, etc., as keys.

See https://docs.kwil.com/docs/extensions/event-listeners for more information on kwil listeners.

### [ingest_resolution](internal/extensions/resolutions/ingest_resolution)

Implements the consensus mechanisms. It is used by `pagination_poll_listener` to resolve data. For example, it says how to serialize data and what to do when consensus is reached.

See https://docs.kwil.com/docs/extensions/resolutions for more information on kwil resolutions.

### [demo-contract](./examples/demo-contract)

Provides a simple kuneiform file demonstrating how we can use the `logstore_listener` and `ingest_resolution` to fetch data from the Log Store into a kwil contract.

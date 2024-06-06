## Run it

### Pre-requisites

- `Taskfile` [Install Instructions](https://taskfile.dev/installation)
- `Docker Compose` [Install Instructions](https://docs.docker.com/compose)
- `node >= v20`
- `streamr-cli` and basic knowledge of [how to use it](https://docs.streamr.network/usage/cli-tool).
  ```shell
   pnpm install -g @streamr/cli-tools@latest
   ```
- `kwil-cli`, `kwil-admin` and a basic knowledge of [how to use it](https://docs.kwil.com/docs/kwil-cli/installation)
  ```shell
  # install both kwil-cli and kwil-admin
  task kwil-binaries
  ```
- Some tokens in your polygon wallet to deploy a stream

### Steps to test a single node

1. Create the demo stream (case the network is unstable, you might prefer using the [web interface](https://streamr.network/hub/streams/new))
    ```bash
   # If your wallet is not configured, remember to add `--private-key <PRIVATE_KEY>` flag
   streamr stream create /kwil-demo
    ```

2. Make the stream public (or via web UI, too)
    ```bash
   streamr stream grant-permission /kwil-demo public subscribe
    ```

3. Adjust the configuration for the Log Store to start tracking the stream.  
   For a Single Node, an adopt the Log Store configuration [here](./examples/single-node/logstore-config.json)
   ```
     "trackedStreams": [
         {
             "id": "<your_address>/kwil-demo",
             "partitions": 1
         }
     ]
   ```

4. Adjust the configuration of the oracle extension.  
   For a Single Node, an adopt the Log Store configuration [here](./examples/single-node/logstore-config.json)
   ```toml
   stream_id = "<your_address>/kwil-demo"
   ```

5. Start the docker services
   ```bash
   docker compose -f ./docker-compose.yaml up -d
   ```

6. Please ensure to you have configured your Kwil CLI, and Private Key
   ```bash
   ./.build/kwil-cli configure
    Kwil RPC URL: (leave empty for default)
    Kwil Chain ID: (leave empty to trust a server-provided value)
    Private Key: (ECDSA wallet private key without "0x" eg. bb00000000000000000000000000000000000000000000000000000000000001)
    ```

7. Deploy the demo schema
   ```bash
   ./.build/kwil-cli database deploy -p=./examples/demo-contract/demo.kf --name=demo --sync
    ```

8. Publish a message to the stream
    ```bash
   # If your wallet is not configured, remember to add `--private-key <PRIVATE_KEY>` flag
   streamr stream publish /kwil-demo
   # then type in some messages in JSON format, such as {"hello": "world"}
    ```
   
   Note: New or rarely used streams might have increased startup time before the Log Store node considers it ready to receive messages. If you want to see if it is receiving messages already, you can check the logs of the Log Store node. You should see something like this:
   ```
   stream <your_address>/kwil-demo is ready
   starting log store oracle for stream <your_address>/kwil-demo
   ```

9. (After 2 minutes) Call an action to get data from kwil node
    ```bash
    ./.build/kwil-cli database call -a=get_data -n=demo
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

// package logstore_listener implements an listener that queries a logstore node for new data within a specific stream
// and ingests it into databases with the help of the ingest resolution,
// looking for datasets with `log_store_ingest($data)` procedure available.
package logstore_listener

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/kwilteam/kwil-db/common"
	"github.com/kwilteam/kwil-db/core/crypto"
	"github.com/kwilteam/kwil-db/core/crypto/auth"
	"github.com/kwilteam/kwil-db/extensions/listeners"
	"github.com/usherlabs/kwil-ls-oracle/internal/extensions/resolutions/ingest_resolution"
	"github.com/usherlabs/kwil-ls-oracle/internal/logstore_client"
	"github.com/usherlabs/kwil-ls-oracle/internal/paginated_poll_listener"
)

const ListenerName = "logstore-oracle"

// use golang's init function, which runs before main, to register the extension
// see more here: https://www.digitalocean.com/community/tutorials/understanding-init-in-go
func init() {
	// register the listener with the name "logstore-oracle"
	err := listeners.RegisterListener(ListenerName, Start)
	if err != nil {
		panic(err)
	}
}

func Start(ctx context.Context, service *common.Service, eventstore listeners.EventStore) error {
	config := LogStoreListenerConfig{}
	// get the listener config
	listenerConfig, ok := service.ExtensionConfigs[ListenerName]
	if !ok {
		service.Logger.Info("no logstore_oracle configuration found, so it will not start")
		return nil // no configuration, so we don't start the oracle
	}

	err := config.setConfig(listenerConfig)
	if err != nil {
		return fmt.Errorf("failed to set config: %w", err)
	}

	privateKey, err := crypto.Secp256k1PrivateKeyFromHex(config.PrivateKey)
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}
	signer := auth.EthPersonalSigner{
		Key: *privateKey,
	}

	// create a new LogStoreClient
	client := logstore_client.NewLogStoreClient(config.NodeEndpoint, signer)

	// create a new LogStorePoller
	poller := NewLogStorePoller(*client, config.StreamId)

	// every 1 minute
	logStoreKeying := NewLogStoreKeying(NewLogStoreKeyingOptions{
		OverheadDelay:     config.OverheadDelay,
		StreamId:          config.StreamId,
		Client:            *client,
		StartingTimestamp: config.StartingTimestamp,
		CronExprStr:       config.CronSchedule,
	})

	// update the ingest resolution with the lookup schemas
	ingest_resolution.LogStoreIngestResolution.ContractSelectors = ingest_resolution.LookupSchemaToSelectors(config.LookupSchemas)

	// create a new PaginatedPoller
	paginatedPoller := paginated_poll_listener.PaginatedPoller[*ingest_resolution.LogStoreIngestDataResolution]{
		PollerService:    poller,
		KeyingService:    logStoreKeying,
		IngestResolution: *ingest_resolution.LogStoreIngestResolution,
	}

	// When the log store node has just started, there's a chance that the node hasn't connected to
	// any node making the stream available yet. To avoid this, we try to query for the ready state using the client.
	// We try it 20 times, with a 30 seconds timeout each.
	//
	// After this timeout, we still make the oracle run as normal. The rationale is that there might be no active publisher yet.
	// Then it's safe to say that there's no data in the stream yet. If a publisher starts later, we will be able to catch up.

	trial := 0
	ready := false
	for trial < 20 {
		service.Logger.Info(fmt.Sprintf("checking for stream %s readiness, trial %d/20", config.StreamId, trial+1))
		ready, err = client.IsPartitionReady(config.StreamId, 0)
		if err != nil {
			service.Logger.Warn(fmt.Sprintf("retrying... failed to connect to LS Node readiness check: %v", err))
			time.Sleep(1 * time.Second)
			continue
		}

		if ready {
			service.Logger.Info(fmt.Sprintf("stream %s is ready", config.StreamId))
			break
		} else {
			service.Logger.Warn(fmt.Sprintf("stream %s is not ready yet", config.StreamId))
		}

		trial++
	}

	if !ready {
		service.Logger.Warn(fmt.Sprintf("no publisher detected for stream %s, but will still run the oracle", config.StreamId))
	}

	service.Logger.Info(fmt.Sprintf("starting logstore oracle for stream %s", config.StreamId))

	// start the paginated poller
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(5 * time.Second):
			err = paginatedPoller.Run(ctx, service, eventstore)
			if err != nil {
				service.Logger.Warn(fmt.Sprintf("failed to run paginated poller: %v", err))
			}
		}
	}
}

type LogStoreListenerConfig struct {
	StreamId     string `json:"stream_id"`
	NodeEndpoint string `json:"node_endpoint"`
	// defaults to 1 minute
	OverheadDelay     time.Duration `json:"overhead_delay"`
	StartingTimestamp *int64        `json:"starting_timestamp"`
	CronSchedule      string        `json:"cron_schedule"`
	PrivateKey        string        `json:"private_key"`
	LookupSchemas     []string      `json:"lookup_schemas"`
}

func (c *LogStoreListenerConfig) setConfig(config map[string]string) error {
	streamId, ok := config["stream_id"]
	if !ok {
		return fmt.Errorf("missing streamId")
	}
	c.StreamId = streamId

	nodeEndpoint, ok := config["node_endpoint"]
	if !ok {
		return fmt.Errorf("missing nodeEndpoint")
	}
	c.NodeEndpoint = nodeEndpoint

	overheadDelay, ok := config["overhead_delay"]
	if !ok {
		c.OverheadDelay = time.Minute
	} else {
		overheadDelayDuration, err := time.ParseDuration(overheadDelay)
		if err != nil {
			return fmt.Errorf("failed to parse overheadDelay: %w", err)
		}
		c.OverheadDelay = overheadDelayDuration
	}

	cronSchedule, ok := config["cron_schedule"]
	if !ok {
		return fmt.Errorf("missing cronSchedule")
	}

	c.CronSchedule = cronSchedule

	startingTimestamp, ok := config["starting_timestamp"]
	if !ok {
		c.StartingTimestamp = nil
	} else {
		startingTimestampInt, err := strconv.ParseInt(startingTimestamp, 10, 64)
		if err != nil {
			return fmt.Errorf("failed to parse startingTimestamp: %w", err)
		}
		c.StartingTimestamp = &startingTimestampInt
	}

	privateKey, ok := config["private_key"]
	if !ok {
		return fmt.Errorf("missing private_key")
	}
	c.PrivateKey = privateKey

	lookupSchemas, ok := config["lookup_schemas"]
	if !ok {
		return fmt.Errorf("missing lookup_schemas")
	}
	c.LookupSchemas = strings.Split(lookupSchemas, ",")

	return nil
}

package paginated_poll_listener

import (
	"context"
	"fmt"
	"github.com/kwilteam/kwil-db/common"
	"github.com/kwilteam/kwil-db/core/log"
	"github.com/kwilteam/kwil-db/extensions/listeners"
	"github.com/usherlabs/kwil-ls-oracle/internal/extensions/resolutions/ingest_resolution"
)

type PaginatedPoller struct {
	PollerService    PollerService
	KeyingService    KeyingService
	IngestResolution ingest_resolution.IngestResolution
}

type PollerService interface {
	// GetData gets the data from the service from the given key range. FROM (inclusive) and TO (exclusive)
	GetData(from, to int64) ([]interface{}, error)
}

// KeyingService helps to get the starting key, current key, key after and key before.
// Key here means the key of the data that we are processing, it could be a block number, a timestamp, etc.
// For now it's an int64, but it could be any type that can be compared.
type KeyingService interface {
	GetStartingKey() (int64, error)
	GetCurrentKey() (int64, error)
	GetKeyAfter(key int64) (int64, error)
	GetKeyBefore(key int64) (int64, error)
}

func (p *PaginatedPoller) Run(ctx context.Context, service *common.Service, eventstore listeners.EventStore) error {
	lastProcessedKeyRef, err := getLastStoredKey(ctx, eventstore)
	if err != nil {
		return fmt.Errorf("failed to get last stored key: %w", err)
	}
	lastProcessedKey := int64(0)
	if lastProcessedKeyRef != nil {
		lastProcessedKey = *lastProcessedKeyRef
	}

	startingKeyRef, err := getFirstStoredKey(ctx, eventstore)
	if err != nil {
		return fmt.Errorf("failed to get starting key: %w", err)
	}

	var startingKey int64
	if startingKeyRef == nil {
		// starting key should not change, that's why we store it in the kv store
		startingKey, err = p.KeyingService.GetStartingKey()
		if err != nil {
			return fmt.Errorf("failed to get starting key: %w", err)
		}
		err = setFirstStoredKey(ctx, eventstore, startingKey)
		if err != nil {
			return fmt.Errorf("failed to set first key: %w", err)
		}
	} else {
		startingKey = *startingKeyRef
	}

	if startingKey > lastProcessedKey {
		lastProcessedKey = startingKey
	}

	currentKey, err := p.KeyingService.GetCurrentKey()
	if err != nil {
		return fmt.Errorf("failed to get current key: %w", err)
	}
	nextKey, err := p.KeyingService.GetKeyAfter(lastProcessedKey)
	if err != nil {
		return fmt.Errorf("failed to get next key: %w", err)
	}

	if lastProcessedKey > nextKey {
		return fmt.Errorf("starting key is greater than the last confirmed key")
	}

	// we will now sync all data from the starting key to the current key in chunks
	for {
		if lastProcessedKey >= currentKey {
			break
		}

		// get the next key chunk. if it is greater than the current key,
		// we will set it to the current key
		nextKey, err := p.KeyingService.GetKeyAfter(lastProcessedKey)
		if err != nil {
			return fmt.Errorf("failed to get next key: %w", err)
		}

		toKey := min(nextKey, currentKey)

		err = p.retrieveAndProcessData(ctx, lastProcessedKey, toKey, eventstore, service.Logger)
		if err != nil {
			return fmt.Errorf("failed to process events: %w", err)
		}

		lastProcessedKey = toKey
	}

	// set the last key processed by the listener
	err = setLastStoredKey(ctx, eventstore, lastProcessedKey)

	if err != nil {
		return fmt.Errorf("failed to set last key: %w", err)
	}

	return nil
}

// retrieveAndProcessData will process all data from the PollerService from the given key range.
func (p *PaginatedPoller) retrieveAndProcessData(ctx context.Context, from, to int64, eventstore listeners.EventStore, logger log.SugaredLogger) error {
	data, err := p.PollerService.GetData(from, to)
	if err != nil {
		return fmt.Errorf("failed to get data: %w", err)
	}

	ingestDataResolution := ingest_resolution.IngestDataResolution{
		Data: data,
	}

	encodedResolutionResult, err := ingestDataResolution.MarshalBinary()

	if err != nil {
		return fmt.Errorf("failed to marshal resolution: %w", err)
	}

	// broadcast the resolution to the network
	err = eventstore.Broadcast(ctx, p.IngestResolution.ResolutionName, encodedResolutionResult)
	if err != nil {
		return err
	}

	// process data
	return nil
}

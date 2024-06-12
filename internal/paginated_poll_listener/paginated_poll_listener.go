package paginated_poll_listener

import (
	"context"
	"fmt"
	"github.com/kwilteam/kwil-db/common"
	"github.com/kwilteam/kwil-db/core/log"
	"github.com/kwilteam/kwil-db/extensions/listeners"
	"github.com/usherlabs/kwil-ls-oracle/internal/extensions/resolutions/ingest_resolution"
)

type PaginatedPoller[T ingest_resolution.IngestDataResolution] struct {
	PollerService    PollerService[T]
	KeyingService    KeyingService
	IngestResolution ingest_resolution.IngestResolution[T]
}

type PollerService[T ingest_resolution.IngestDataResolution] interface {
	// GetData gets the data from the service from the given key range. FROM (inclusive) and TO (exclusive)
	GetData(from, to int64) (*T, error)
	// EmptyResolutionSize returns the size of the empty resolution
	EmptyResolutionSize() int
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

func (p *PaginatedPoller[T]) Run(ctx context.Context, service *common.Service, eventstore listeners.EventStore) error {
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

	currentKey, err := p.KeyingService.GetCurrentKey()
	if err != nil {
		return fmt.Errorf("failed to get current key: %w", err)
	}

	var startingKey int64
	if startingKeyRef == nil {
		// starting key should not change, that's why we store it in the kv store
		startingKey, err = p.KeyingService.GetStartingKey()
		if err != nil {
			return fmt.Errorf("failed to get starting key: %w", err)
		}

		// if starting key is 0, we set it as the current key.
		// starging key = 0 means there is no data in the system yet.
		if startingKey == 0 {
			startingKey = currentKey
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

	// ending key is the key before the current key
	// e.g., for a service that processes every 10 keys. current key = 102, last processed key = 80
	// - The ending key would be 100;
	// - We expect to process data from 80 to 100, as 100 forward batch is ongoing.
	endingKey, err := p.KeyingService.GetKeyBefore(currentKey)
	if err != nil {
		return fmt.Errorf("failed to get ending key: %w", err)
	}

	var nextKey int64
	// we will now process the data from the last processed key to the ending key
	for {
		nextKey, err = p.KeyingService.GetKeyAfter(lastProcessedKey)

		// should never happen
		if lastProcessedKey > nextKey {
			return fmt.Errorf("starting key is greater than the last confirmed key")
		}

		if err != nil {
			return fmt.Errorf("failed to get next key: %w", err)
		}

		// if nextKey reached the end, we will break the loop, ending the process
		if nextKey > endingKey {
			break
		}

		processErrors := p.retrieveAndProcessData(ctx, lastProcessedKey, nextKey, eventstore, service.Logger)
		if processErrors != nil {
			// if it's not partial, we will return the errors, as this might need to be retried
			if !processErrors.PartiallyProcessed {
				return fmt.Errorf("failed to process data: %w", processErrors.Errors[0])
			}
			// if it's just partial, we will continue to process the next key
		}

		lastProcessedKey = nextKey
	}

	// set the last key processed by the listener
	err = setLastStoredKey(ctx, eventstore, lastProcessedKey)

	if err != nil {
		return fmt.Errorf("failed to set last key: %w", err)
	}

	return nil
}

type ProcessErrors[T any] struct {
	Errors             []error
	PartiallyProcessed bool
	UnprocessedData    []*T
}

// retrieveAndProcessData will process all data from the PollerService from the given key range.
// it returns errors if there are any, and also the unprocessed data
func (p *PaginatedPoller[T]) retrieveAndProcessData(
	ctx context.Context,
	from, to int64,
	eventstore listeners.EventStore,
	logger log.SugaredLogger,
) *ProcessErrors[T] {
	errors := ProcessErrors[T]{
		PartiallyProcessed: false,
		UnprocessedData:    nil,
	}

	ingestDataResolution, err := p.PollerService.GetData(from, to)
	if err != nil {
		errors.Errors = append(errors.Errors, fmt.Errorf("failed to get data: %w", err))
		return &errors
	}

	// if data is nil, we will not process it
	if ingestDataResolution == nil {
		logger.Debug(fmt.Sprintf("no data from %d to %d", from, to))
		return nil
	}

	// btree version 4 maximum size for index
	MaxResolutionSize := 2704
	// this is just a safety measure, because the data wrapper also has some size
	emptyOverheadSize := p.PollerService.EmptyResolutionSize()
	// 54 added by debugging the size of an event during this implementation
	empiricalOverheadSize := 54
	AdoptedMaxSize := MaxResolutionSize - emptyOverheadSize - empiricalOverheadSize

	encodedResolutionResults, chunkedResolutions, err := (*ingestDataResolution).MarshalIntoChunks(AdoptedMaxSize)

	if err != nil {
		errors.Errors = append(errors.Errors, fmt.Errorf("failed to marshal resolution: %w", err))
		return &errors
	}

	for i := 0; i < len(encodedResolutionResults); i++ {
		err = eventstore.Broadcast(ctx, p.IngestResolution.ResolutionName, encodedResolutionResults[i])

		if err != nil {
			// we already broadcasted some, we should append the errors
			errors.Errors = append(errors.Errors, fmt.Errorf("failed to broadcast resolution: %w", err))
			errors.PartiallyProcessed = true
			resolution := chunkedResolutions[i].(T)
			errors.UnprocessedData = append(errors.UnprocessedData, &resolution)
		}
	}

	logger.Info(fmt.Sprintf("broadcasted resolution %s from %d to %d", p.IngestResolution.ResolutionName, from, to))

	// if got more than 1 error, we return the errors
	if len(errors.Errors) > 0 {
		logger.Warn("failed to process data: %v", errors.Errors)
		return &errors
	} else {
		return nil
	}
}

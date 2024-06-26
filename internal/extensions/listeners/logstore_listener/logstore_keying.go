package logstore_listener

import "github.com/gitploy-io/cronexpr"

import (
	"github.com/usherlabs/kwil-ls-oracle/internal/logstore_client"
	"time"
)

// LogStoreKeying is a keying service for the logstore listener.
// it should implement the [paginated_poll_listener.KeyingService] interface.
type LogStoreKeying struct {
	client            logstore_client.LogStoreClient
	streamId          string
	startingTimestamp *int64 // optional
	cronExpr          cronexpr.Schedule
	overheadDelay     time.Duration
}

type NewLogStoreKeyingOptions struct {
	Client            logstore_client.LogStoreClient
	StreamId          string
	StartingTimestamp *int64
	CronExprStr       string
	OverheadDelay     time.Duration
}

func NewLogStoreKeying(options NewLogStoreKeyingOptions) *LogStoreKeying {
	cronExpr, err := cronexpr.Parse(options.CronExprStr)
	if err != nil {
		panic(err)
	}

	return &LogStoreKeying{
		client:            options.Client,
		streamId:          options.StreamId,
		startingTimestamp: options.StartingTimestamp,
		cronExpr:          *cronExpr,
		overheadDelay:     options.OverheadDelay,
	}
}

// GetStartingKey gets the starting key for the logstore listener.
func (l *LogStoreKeying) GetStartingKey() (int64, error) {
	// if starting timestamp is provided, return it
	if l.startingTimestamp != nil {
		return *l.startingTimestamp, nil
	}
	// else, we consider the first message timestamp in the stream
	return l.client.GetFirstMessageTimestamp(l.streamId)
}

// GetCurrentKey gets the current key for the logstore listener.
// it should return the current timestamp in UTC.
// Overhead delay is added per configuration, so we can say that we only validate data that is at least overheadDelay old.
func (l *LogStoreKeying) GetCurrentKey() (int64, error) {
	// let's return current timestamp in UTC from time
	// alternatively we may switch it to timestamp in the future
	// overhead delay is added per configuration
	return time.Now().Add(-l.overheadDelay).UnixMilli(), nil
}

// GetKeyAfter gets the key after the given key for the logstore listener.
func (l *LogStoreKeying) GetKeyAfter(key int64) (int64, error) {
	// convert from unix timestamp to time
	keyTime := time.UnixMilli(key)

	return l.cronExpr.Next(keyTime).UnixMilli(), nil
}

// GetKeyBefore gets the key before the given key for the logstore listener.
func (l *LogStoreKeying) GetKeyBefore(key int64) (int64, error) {
	// convert from unix timestamp to time
	keyTime := time.UnixMilli(key)

	// we get the prev from the next, because prev considers we're sitting on the key
	// i.e., for a cron that runs every minute 00:00, 01:00, 02:00,
	// if current is 01:30, we want key before to be 01:00
	// so we get next: 02:00, and then prev: 01:00
	// otherwise if we try to get prev directly, we would get 00:00
	next := l.cronExpr.Next(keyTime)
	return l.cronExpr.Prev(next).UnixMilli(), nil
}

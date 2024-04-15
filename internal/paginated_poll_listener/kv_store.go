package paginated_poll_listener

import (
	"context"
	"encoding/binary"
	"fmt"
	"github.com/kwilteam/kwil-db/extensions/listeners"
)

var (
	firstKeyKey = []byte("fk")
	// lastKeyKey is the key used to store the last key processed by the listener
	lastKeyKey = []byte("lk")
)

// convertIntToBytes converts an int64 to a byte slice
func convertIntToBytes(i int64) []byte {
	bytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(bytes, uint64(i))
	return bytes
}

// convertBytesToInt converts a byte slice to an int64
func convertBytesToInt(b []byte) int64 {
	return int64(binary.LittleEndian.Uint64(b))
}

// getStoredKey gets a key processed and stored by the KV store
func getStoredKey(ctx context.Context, eventstore listeners.EventStore, key []byte) (*int64, error) {
	storedKey, err := eventstore.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get key: %w", err)
	}

	if len(storedKey) == 0 {
		return nil, nil
	}

	keyInt := convertBytesToInt(storedKey)
	return &keyInt, nil
}

// setStoredKey sets a key stored by the KV store
func setStoredKey(ctx context.Context, eventstore listeners.EventStore, key []byte, value int64) error {
	valueBytes := convertIntToBytes(value)

	err := eventstore.Set(ctx, key, valueBytes)
	if err != nil {
		return fmt.Errorf("failed to set key: %w", err)
	}
	return nil
}

func getFirstStoredKey(ctx context.Context, eventstore listeners.EventStore) (*int64, error) {
	return getStoredKey(ctx, eventstore, firstKeyKey)
}

func setFirstStoredKey(ctx context.Context, eventstore listeners.EventStore, timestamp int64) error {
	return setStoredKey(ctx, eventstore, firstKeyKey, timestamp)
}

func getLastStoredKey(ctx context.Context, eventstore listeners.EventStore) (*int64, error) {
	return getStoredKey(ctx, eventstore, lastKeyKey)
}

func setLastStoredKey(ctx context.Context, eventstore listeners.EventStore, timestamp int64) error {
	return setStoredKey(ctx, eventstore, lastKeyKey, timestamp)
}

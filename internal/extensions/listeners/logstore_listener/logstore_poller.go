package logstore_listener

import "github.com/usherlabs/kwil-ls-oracle/internal/logstore_client"

type LogStorePoller struct {
	client   logstore_client.LogStoreClient
	streamId string
}

func NewLogStorePoller(client logstore_client.LogStoreClient, streamId string) *LogStorePoller {
	return &LogStorePoller{client: client, streamId: streamId}
}

// GetData gets the data from the service from the given key range. FROM (inclusive) and TO (exclusive)
func (l *LogStorePoller) GetData(from, to int64) ([]interface{}, error) {
	messages, err := l.client.QueryAllPartitions(l.streamId, from, to-1)

	var messagesContent []interface{}
	if err != nil {
		return nil, err
	}

	for _, message := range messages {
		messagesContent = append(messagesContent, message.Content)
	}

	return messagesContent, nil
}

package logstore_listener

import (
	"encoding/json"
	"github.com/usherlabs/kwil-ls-oracle/internal/extensions/resolutions/ingest_resolution"
	"github.com/usherlabs/kwil-ls-oracle/internal/logstore_client"
	"github.com/usherlabs/kwil-ls-oracle/internal/paginated_poll_listener"
	"strconv"
	"strings"
)

// LogStorePoller is a poller service for the logstore listener.
// it should implement the [paginated_poll_listener.PollerService] interface.
type LogStorePoller struct {
	client   logstore_client.LogStoreClient
	streamId string
}

var _ paginated_poll_listener.PollerService[*ingest_resolution.LogStoreIngestDataResolution] = (*LogStorePoller)(nil)

func NewLogStorePoller(client logstore_client.LogStoreClient, streamId string) *LogStorePoller {
	return &LogStorePoller{client: client, streamId: streamId}
}

// GetData gets the data from the service from the given key range. FROM (inclusive) and TO (exclusive)
func (l *LogStorePoller) GetData(from, to int64) (**ingest_resolution.LogStoreIngestDataResolution, error) {
	messages, err := l.client.QueryAllPartitions(l.streamId, from, to-1)

	if err != nil {
		return nil, err
	}

	// if there are no messages, return nil
	if len(messages) == 0 {
		return nil, nil
	}

	ingestMessages := make([]ingest_resolution.LogStoreIngestMessage, 0, len(messages))
	for _, message := range messages {
		// json encode content
		strContent := ""
		if message.Content != nil {
			content, err := json.Marshal(message.Content)
			if err != nil {
				return nil, err
			}
			strContent = string(content)
		}

		idComponents := []string{
			strconv.Itoa(int(message.Timestamp)),
			strconv.Itoa(message.SequenceNumber),
			strconv.Itoa(message.StreamPartition),
		}
		id := strings.Join(idComponents, "_")

		ingestMessages = append(ingestMessages, ingest_resolution.LogStoreIngestMessage{
			Id:        id,
			Content:   strContent,
			Timestamp: uint(message.Timestamp),
		})
	}

	data := &ingest_resolution.LogStoreIngestDataResolution{
		Messages: ingestMessages,
	}

	return &data, nil
}

var emptyResolution = &ingest_resolution.LogStoreIngestDataResolution{
	Messages: make([]ingest_resolution.LogStoreIngestMessage, 0),
}
var encodedEmptyResolution, _ = json.Marshal(emptyResolution)

func (l *LogStorePoller) EmptyResolutionSize() int {
	return len(encodedEmptyResolution)
}

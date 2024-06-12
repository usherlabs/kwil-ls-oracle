package ingest_resolution

import (
	"github.com/kwilteam/kwil-db/core/types/serialize"
	"math/big"
	"strconv"
)

type LogStoreIngestMessage struct {
	Content   string
	Timestamp uint
}

func (m *LogStoreIngestMessage) MarshalBinary() ([]byte, error) {
	return serialize.Encode(m)
}

func (m *LogStoreIngestMessage) UnmarshalBinary(rawData []byte) error {
	return serialize.DecodeInto(rawData, m)
}

type LogStoreIngestDataResolution struct {
	Messages []LogStoreIngestMessage
}

func (r *LogStoreIngestDataResolution) NewData() IngestDataResolution {
	return &LogStoreIngestDataResolution{}
}

func (r *LogStoreIngestDataResolution) MarshalBinary() ([]byte, error) {
	return serialize.Encode(r)
}

func (r *LogStoreIngestDataResolution) UnmarshalBinary(rawData []byte) error {
	return serialize.DecodeInto(rawData, &r)
}

func (r *LogStoreIngestDataResolution) MarshalIntoChunks(maxChunkSize int) ([][]byte, []IngestDataResolution, error) {
	binaryData, err := r.MarshalBinary()
	if err != nil {
		return nil, nil, err
	}

	binarySize := len(binaryData)

	var chunks [][]byte
	var rs []IngestDataResolution

	if binarySize < maxChunkSize {
		chunks = append(chunks, binaryData)
		rs = append(rs, r)
		return chunks, rs, nil
	}

	// split the data into chunks
	splitChunks := r.split(2)

	for _, rawChunk := range splitChunks {
		newChunks, newRs, err := rawChunk.MarshalIntoChunks(maxChunkSize)
		if err != nil {
			return nil, nil, err
		}
		chunks = append(chunks, newChunks...)
		rs = append(rs, newRs...)
	}

	return chunks, rs, nil
}

func (r *LogStoreIngestDataResolution) split(numberOfChunks int) []*LogStoreIngestDataResolution {
	chunkSize := (len(r.Messages) + numberOfChunks - 1) / numberOfChunks
	chunks := make([]*LogStoreIngestDataResolution, 0, numberOfChunks)
	for i := 0; i < len(r.Messages); i += chunkSize {
		end := i + chunkSize
		if end > len(r.Messages) {
			end = len(r.Messages)
		}
		resolution := &LogStoreIngestDataResolution{
			Messages: r.Messages[i:end],
		}
		chunks = append(chunks, resolution)
	}
	return chunks
}

func (r *LogStoreIngestDataResolution) GetArgs() [][]*string {
	var argsSet [][]*string
	for _, message := range r.Messages {
		var args []*string
		args = append(args, &message.Content)
		tsString := strconv.Itoa(int(message.Timestamp))
		args = append(args, &tsString)
		argsSet = append(argsSet, args)
	}

	return argsSet
}

var LogStoreIngestResolution = &IngestResolution[*LogStoreIngestDataResolution]{
	RefundThreshold:       big.NewRat(1, 3),
	ConfirmationThreshold: big.NewRat(2, 3),
	// aprox 1 hour, assuming 6s block time
	ExpirationPeriod: 600,
	ResolutionName:   "log_store_ingest",
}

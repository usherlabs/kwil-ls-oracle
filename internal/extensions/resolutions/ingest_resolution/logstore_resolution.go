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
	return serialize.DecodeInto(rawData, r)
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

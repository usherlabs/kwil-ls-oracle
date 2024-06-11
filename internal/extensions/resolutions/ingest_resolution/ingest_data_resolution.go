package ingest_resolution

type IngestDataResolution interface {
	NewData() IngestDataResolution
	// MarshalBinary marshals the resolution into a binary format.
	// It needs to be deterministic and unique for each resolution.
	MarshalBinary() ([]byte, error)
	// UnmarshalBinary unmarshals the resolution from a binary format.
	UnmarshalBinary(rawData []byte) error
	// GetArgs converts the resolution into a list of arguments to be used in multiple procedure calls.
	// currently, kwil only accepts string or nil arguments
	GetArgs() [][]*string
	// MarshalIntoChunks converts the resolution into a list of chunks with a max size for each chunk
	MarshalIntoChunks(maxChunkSize int) ([][]byte, []IngestDataResolution, error)
}

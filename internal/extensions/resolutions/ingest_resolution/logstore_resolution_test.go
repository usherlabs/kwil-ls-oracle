package ingest_resolution

import (
	"reflect"
	"testing"
)

func TestMarshalBinary(t *testing.T) {
	testCases := []struct {
		name     string
		messages []LogStoreIngestMessage
	}{
		// Test case with an empty struct
		{
			name:     "Empty struct",
			messages: []LogStoreIngestMessage{},
		},
		// Test case with one message
		{
			name:     "One message",
			messages: []LogStoreIngestMessage{{Timestamp: 1713966823, Content: "Message 1"}},
		},
		// Test case with multiple messages
		{
			name:     "Multiple messages",
			messages: []LogStoreIngestMessage{{Timestamp: 1713966823, Content: "Message 1"}, {Timestamp: 1713966824, Content: "Message 2"}},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			original := LogStoreIngestDataResolution{Messages: testCase.messages}

			// Use MarshalBinary to marshal the original object
			data, err := original.MarshalBinary()
			if err != nil {
				t.Fatalf("Failed to marshal: %s", err)
			}

			// Create a new object and use UnmarshalBinary to unmarshal the data into it
			var unmarshalled LogStoreIngestDataResolution
			err = unmarshalled.UnmarshalBinary(data)
			if err != nil {
				t.Fatalf("Failed to unmarshal: %s", err)
			}

			// The original and unmarshalled object should be the same
			if !reflect.DeepEqual(original, unmarshalled) {
				t.Errorf("%s: expected %v, got %v", testCase.name, original, unmarshalled)
			}
		})
	}
}

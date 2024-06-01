package logstore_client

import (
	"encoding/json"
	"fmt"
	"github.com/kwilteam/kwil-db/core/crypto/auth"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

type LogStoreClient struct {
	endpoint string
	signer   auth.EthPersonalSigner
}

func NewLogStoreClient(endpoint string, signer auth.EthPersonalSigner) *LogStoreClient {
	return &LogStoreClient{endpoint: endpoint, signer: signer}
}

func (c *LogStoreClient) GetCurrentBlockHeight() (int64, error) {
	return 0, fmt.Errorf("not implemented")
}

// FetchMessages fetches messages from the log store using a request
func (c *LogStoreClient) FetchMessages(req *http.Request) ([]JSONStreamMessage, error) {
	authHeader, err := createAuthHeader(c.signer)
	if err != nil {
		return nil, err
	}

	req.Header.Add("authorization", authHeader)

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	// parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return decodeStreamMessageResponse(body)
}

func (c *LogStoreClient) GetFirstMessageTimestamp(streamId string) (int64, error) {
	// http://<endpoint>/stores/:id/data/partitions/:partition/last?count=-1 (count=-1 means get the first message)

	encodedStreamId := url.PathEscape(streamId)
	req, err := http.NewRequest("GET", c.endpoint+"/stores/"+encodedStreamId+"/data/partitions/0/last", nil)
	if err != nil {
		panic(err)
	}

	q := req.URL.Query()
	q.Add("count", "-1")
	req.URL.RawQuery = q.Encode()

	streamMessageResponse, err := c.FetchMessages(req)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch messages: %w", err)
	}

	// if there's no message, return 0
	if len(streamMessageResponse) == 0 {
		return 0, nil
	}

	return streamMessageResponse[0].Timestamp, nil
}

func (c *LogStoreClient) GetLatestMessageTimestamp(streamId string) (int64, error) {
	// http://<endpoint>/stores/:id/data/partitions/:partition/last?count=1 (count=1 means get the last message)

	encodedStreamId := url.PathEscape(streamId)
	req, err := http.NewRequest("GET", c.endpoint+"/stores/"+encodedStreamId+"/data/partitions/0/last", nil)
	if err != nil {
		panic(err)
	}

	q := req.URL.Query()
	q.Add("count", "1")
	req.URL.RawQuery = q.Encode()

	streamMessageResponse, err := c.FetchMessages(req)

	if err != nil {
		return 0, fmt.Errorf("failed to fetch messages: %w", err)
	}

	return streamMessageResponse[0].Timestamp, nil
}

func (c *LogStoreClient) GetStreamPartitionCount(streamId string) (int, error) {
	return 0, fmt.Errorf("not implemented")
}

func (c *LogStoreClient) QueryAllPartitions(streamId string, from, to int64) ([]JSONStreamMessage, error) {
	return c.QueryRange(streamId, from, to, 0)
}

func (c *LogStoreClient) QueryRange(streamId string, from, to int64, partition int) ([]JSONStreamMessage, error) {
	// http://<endpoint>/stores/:id/data/partitions/:partition/range?from=:from&to=:to
	encodedStreamId := url.PathEscape(streamId)
	req, err := http.NewRequest("GET", c.endpoint+"/stores/"+encodedStreamId+"/data/partitions/"+strconv.Itoa(partition)+"/range", nil)
	if err != nil {
		panic(err)
	}

	q := req.URL.Query()
	q.Add("fromTimestamp", strconv.FormatInt(from, 10))
	q.Add("toTimestamp", strconv.FormatInt(to, 10))
	req.URL.RawQuery = q.Encode()

	return c.FetchMessages(req)
}

// {
//    "messages": [
//        {
//            "streamId": "0xd37dc4d7e2c1bdf3edd89db0e505394ea69af43d/kwil-demo",
//            "streamPartition": 0,
//            "timestamp": 1717251456911,
//            "sequenceNumber": 0,
//            "publisherId": "0xd37dc4d7e2c1bdf3edd89db0e505394ea69af43d",
//            "msgChainId": "SSMpFE1J9BcHiq5shY3Q",
//            "messageType": 27,
//            "contentType": 0,
//            "encryptionType": 0,
//            "content": 1,
//            "signatureType": 1,
//            "signature": "6e5bc0cbb8f8a6e3af351758e179f5073b15d7591ce5923f552727f4d076e53b1db46fd7c508f89a9c07c3b17612cc961ea5a46ba502bed0d5221c3da59282611c"
//        }
//    ],
//    "metadata": {
//        "hasNext": false,
//        "totalMessages": 1,
//        "type": "metadata"
//    }
//}

type JSONStreamMessage struct {
	StreamId        string      `json:"streamId"`
	StreamPartition int         `json:"streamPartition"`
	Timestamp       int64       `json:"timestamp"`
	SequenceNumber  int         `json:"sequenceNumber"`
	PublisherId     string      `json:"publisherId"`
	MsgChainId      string      `json:"msgChainId"`
	MessageType     int         `json:"messageType"`
	ContentType     int         `json:"contentType"`
	EncryptionType  int         `json:"encryptionType"`
	Content         interface{} `json:"content"`
	SignatureType   int         `json:"signatureType"`
	Signature       string      `json:"signature"`
}

// create decoder from response body
func decodeStreamMessageResponse(body []byte) ([]JSONStreamMessage, error) {
	var response struct {
		Messages []JSONStreamMessage `json:"messages"`
		Metadata struct {
			HasNext       bool   `json:"hasNext"`
			TotalMessages int    `json:"totalMessages"`
			Type          string `json:"type"`
		} `json:"metadata"`
	}

	err := json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	return response.Messages, nil
}

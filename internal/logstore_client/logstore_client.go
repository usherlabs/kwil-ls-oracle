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

	return streamMessageResponse[0].Metadata.Id.Timestamp, nil
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

	return streamMessageResponse[0].Metadata.Id.Timestamp, nil
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

type JSONStreamMessage struct {
	Metadata struct {
		Id struct {
			StreamId        string `json:"streamId"`
			StreamPartition int    `json:"streamPartition"`
			Timestamp       int64  `json:"timestamp"`
			SequenceNumber  int    `json:"sequenceNumber"`
			PublisherId     string `json:"publisherId"`
			MsgChainId      string `json:"msgChainId"`
		} `json:"id"`
		PrevMsgRef     interface{} `json:"prevMsgRef"`
		MessageType    int         `json:"messageType"`
		ContentType    int         `json:"contentType"`
		EncryptionType int         `json:"encryptionType"`
		GroupKeyId     interface{} `json:"groupKeyId"`
		NewGroupKey    interface{} `json:"newGroupKey"`
		Signature      string      `json:"signature"`
	} `json:"metadata"`
	Content interface{} `json:"content"`
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

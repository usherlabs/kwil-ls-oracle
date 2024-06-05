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

/*
 * Path: /stores/:id/partitions/:partition/ready
 * Query parameters:
 * - timeout: in milliseconds (30000 by default). If the node is not ready in this time, it will return false.
 *
 * Response:
 * {"ready":bool}
 */

type JSONPartitionReadyResponse struct {
	Ready bool `json:"ready"`
}

func (c *LogStoreClient) IsPartitionReady(streamId string, partition int) (bool, error) {
	req, err := http.NewRequest("GET", c.endpoint+"/stores/"+url.PathEscape(streamId)+"/partitions/"+strconv.Itoa(partition)+"/ready", nil)
	if err != nil {
		panic(err)
	}

	q := req.URL.Query()
	q.Add("timeout", "30000")
	req.URL.RawQuery = q.Encode()

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return false, fmt.Errorf("failed to check if partition is ready: %w", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read response body: %w", err)
	}

	var response JSONPartitionReadyResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return false, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return response.Ready, nil
}

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

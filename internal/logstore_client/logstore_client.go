package logstore_client

import (
	"encoding/json"
	"fmt"
	"github.com/kwilteam/kwil-db/core/crypto/auth"
	"net/http"
	"strconv"
)

type LogStoreClient struct {
	endpoint string
	signer   auth.Signer
}

func NewLogStoreClient(endpoint string, signer auth.Signer) *LogStoreClient {
	return &LogStoreClient{endpoint: endpoint, signer: signer}
}

func (c *LogStoreClient) GetCurrentBlockHeight() (int64, error) {
	return 0, fmt.Errorf("not implemented")
}

func (c *LogStoreClient) GetFirstMessageTimestamp(streamId string) (int64, error) {
	// http://<endpoint>/stores/:id/data/partitions/:partition/from?fromTimestamp=:fromTimestamp

	encodedStreamId := strconv.Quote(streamId)
	req, err := http.NewRequest("GET", c.endpoint+"/stores/"+encodedStreamId+"/data/partitions/0/from", nil)
	if err != nil {
		panic(err)
	}

	q := req.URL.Query()
	q.Add("fromTimestamp", "0")
	req.URL.RawQuery = q.Encode()

	authToken, err := createAuthHeader(c.signer)
	if err != nil {
		return 0, err
	}

	authHeader := "Basic " + authToken
	req.Header.Add("Authorization", authHeader)

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return 0, err
	}

	defer resp.Body.Close()

	// parse response
	body := make([]byte, 0)
	_, err = resp.Body.Read(body)
	if err != nil {
		panic(err)
	}

	streamMessageResponse, err := decodeStreamMessageResponse(body)

	return strconv.ParseInt(streamMessageResponse[0].Metadata.Id, 10, 64)
}

func (c *LogStoreClient) GetLatestMessageTimestamp(streamId string) (int64, error) {
	return 0, fmt.Errorf("not implemented")
}

func (c *LogStoreClient) GetStreamPartitionCount(streamId string) (int, error) {
	return 0, fmt.Errorf("not implemented")
}

func (c *LogStoreClient) QueryAllPartitions(streamId string, from, to int64) ([]JSONStreamMessage, error) {
	return c.QueryRange(streamId, from, to, 0)
}

func (c *LogStoreClient) QueryRange(streamId string, from, to int64, partition int) ([]JSONStreamMessage, error) {
	// http://<endpoint>/stores/:id/data/partitions/:partition/range?from=:from&to=:to
	req, err := http.NewRequest("GET", c.endpoint+"/stores/"+streamId+"/data/partitions/"+string(partition)+"/range", nil)
	if err != nil {
		panic(err)
	}

	q := req.URL.Query()
	q.Add("from", string(from))
	q.Add("to", string(to))
	req.URL.RawQuery = q.Encode()

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	// parse response
	body := make([]byte, 0)
	_, err = resp.Body.Read(body)
	if err != nil {
		panic(err)
	}

	streamMessageResponse, err := decodeStreamMessageResponse(body)

	return streamMessageResponse, err
}

type JSONStreamMessage struct {
	Metadata struct {
		Id             string `json:"id"`
		PrevMsgRef     string `json:"prevMsgRef"`
		MessageType    string `json:"messageType"`
		ContentType    string `json:"contentType"`
		EncryptionType string `json:"encryptionType"`
		GroupKeyId     string `json:"groupKeyId"`
		NewGroupKey    string `json:"newGroupKey"`
		Signature      string `json:"signature"`
	} `json:"metadata"`
	Content string `json:"content"`
}

// create decoder from response body
func decodeStreamMessageResponse(body []byte) ([]JSONStreamMessage, error) {
	var response struct {
		Messages []JSONStreamMessage `json:"messages"`
	}

	err := json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	return response.Messages, nil
}

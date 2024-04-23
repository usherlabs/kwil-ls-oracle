package logstore_client

import (
	"fmt"
	"github.com/kwilteam/kwil-db/core/crypto"
	"github.com/kwilteam/kwil-db/core/crypto/auth"
	"gotest.tools/assert"
	"testing"
)

func Test_GetFirstMessageTimestamp(t *testing.T) {
	tests := []struct {
		name     string
		prepare  func() (*LogStoreClient, error)
		streamId string
		wantErr  bool
	}{
		{
			name:     "Normal Case",
			streamId: "0xd37dc4d7e2c1bdf3edd89db0e505394ea69af43d/kwil-demo",
			prepare: func() (*LogStoreClient, error) {
				privateKey, err := crypto.Secp256k1PrivateKeyFromHex("0000000000000000000000000000000000000000000000000000000000000022")
				if err != nil {
					return nil, fmt.Errorf("failed to parse private key: %w", err)
				}
				signer := auth.EthPersonalSigner{
					Key: *privateKey,
				}

				return NewLogStoreClient("http://localhost:7773", signer), nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := tt.prepare()
			assert.NilError(t, err)
			got, err := c.GetFirstMessageTimestamp(tt.streamId)
			if tt.wantErr {
				assert.Error(t, err, "expected error but got nil")
				return
			}
			assert.NilError(t, err)
			assert.Assert(t, got > 0)
		})
	}
}

package logstore_client

import (
	"encoding/base64"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/kwilteam/kwil-db/core/crypto/auth"
)

func createAuthHeader(signer auth.EthPersonalSigner) (string, error) {
	user := signer.Identity()
	userStr := hexutil.Encode(user)

	passwordSignature, err := signer.Sign([]byte(userStr))
	if err != nil {
		return "", err
	}
	signatureStr := base64.StdEncoding.EncodeToString(passwordSignature.Signature)

	token := userStr + ":" + signatureStr
	base64Token := base64.StdEncoding.EncodeToString([]byte(token))

	return "basic " + base64Token, nil
}

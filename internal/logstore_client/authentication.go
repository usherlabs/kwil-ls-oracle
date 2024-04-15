package logstore_client

import (
	"github.com/kwilteam/kwil-db/core/crypto/auth"
)

func createAuthHeader(signer auth.Signer) (string, error) {
	user := signer.Identity()
	passwordSignature, err := signer.Sign(user)
	if err != nil {
		return "", err
	}
	token := string(user) + ":" + string(passwordSignature.Signature)

	return token, nil
}

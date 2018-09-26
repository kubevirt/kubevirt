package apiserver

import (
	"crypto/rsa"
	"encoding/json"
	"time"

	"gopkg.in/square/go-jose.v2"

	"github.com/pkg/errors"
)

const (
	// APIPublicKeyConfigMap is the uploadProxy Public key
	APIPublicKeyConfigMap = "cdi-api-public"

	// timeout seconds for each token. 5 minutes
	tokenTimeout = 300
)

// TokenData defines the data in the upload token
type TokenData struct {
	PvcName           string    `json:"pvcName"`
	Namespace         string    `json:"namespace"`
	CreationTimestamp time.Time `json:"creationTimestamp"`
}

// VerifyToken checks the token signature and returns the contents
func VerifyToken(token string, publicKey *rsa.PublicKey) (*TokenData, error) {
	object, err := jose.ParseSigned(token)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to parse token")
	}

	message, err := object.Verify(publicKey)
	if err != nil {
		return nil, errors.Wrap(err, "Token verification failed")
	}

	tokenData := &TokenData{}
	err = json.Unmarshal(message, tokenData)
	if err != nil {
		return nil, errors.Wrap(err, "Error unmarshaling JSON")
	}

	// don't allow expired tokens to be viewed
	start := tokenData.CreationTimestamp.Unix()
	now := time.Now().UTC().Unix()
	diff := now - start
	if diff < 0 {
		diff *= -1
	}
	if diff > tokenTimeout {
		return nil, errors.Errorf("Token expired")
	}

	// If we get here, the message is good
	return tokenData, nil
}

// GenerateToken generates a token from the given parameters
func GenerateToken(pvcName string, namespace string, signingKey *rsa.PrivateKey) (string, error) {
	tokenData := &TokenData{
		Namespace:         namespace,
		PvcName:           pvcName,
		CreationTimestamp: time.Now().UTC(),
	}

	message, err := json.Marshal(tokenData)
	if err != nil {
		return "", errors.Wrap(err, "JSON Marshal failed")
	}

	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.PS512, Key: signingKey}, nil)
	if err != nil {
		return "", errors.Wrap(err, "Error creating JWT signer")
	}

	object, err := signer.Sign(message)
	if err != nil {
		return "", errors.Wrap(err, "Error signing JWT message")
	}

	serialized, err := object.CompactSerialize()
	if err != nil {
		return "", errors.Wrap(err, "Error serializing JWT message")
	}

	return serialized, nil
}

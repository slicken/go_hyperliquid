package hyperliquid

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type PKeyManager struct {
	PrivateKeyStr string
	privateKey    *ecdsa.PrivateKey
	publicKey     *ecdsa.PublicKey
}

func (km *PKeyManager) PublicECDSA() *ecdsa.PublicKey {
	return km.publicKey
}

func (km *PKeyManager) PrivateECDSA() *ecdsa.PrivateKey {
	return km.privateKey
}

func (km *PKeyManager) PublicAddress() common.Address {
	return crypto.PubkeyToAddress(*km.publicKey)
}

func (km *PKeyManager) PublicAddressHex() string {
	return km.PublicAddress().Hex()
}

// NewPKeyManager creates a new PKeyManager instance from a private key string
func NewPKeyManager(privateKey string) (*PKeyManager, error) {
	privKey, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		return nil, err
	}
	publicKey, ok := privKey.Public().(*ecdsa.PublicKey)
	if !ok {
		return nil, err
	}
	return &PKeyManager{privateKey: privKey, publicKey: publicKey, PrivateKeyStr: privateKey}, nil
}

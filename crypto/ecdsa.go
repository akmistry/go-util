package crypto

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/asn1"
	"math/big"
)

type ecdsaSig struct {
	R, S *big.Int
}

func MarshalECSig(r, s *big.Int) ([]byte, error) {
	sig := ecdsaSig{R: r, S: s}
	return asn1.Marshal(sig)
}

func UnmarshalECSig(b []byte) (r, s *big.Int, err error) {
	var sig ecdsaSig
	_, err = asn1.Unmarshal(b, &sig)
	if err != nil {
		return
	}
	return sig.R, sig.S, nil
}

func SignEC(priv *ecdsa.PrivateKey, msg []byte) ([]byte, error) {
	r, s, err := ecdsa.Sign(rand.Reader, priv, msg)
	if err != nil {
		return nil, err
	}
	return MarshalECSig(r, s)
}

func VerifyECSig(pub *ecdsa.PublicKey, msg, sig []byte) bool {
	r, s, err := UnmarshalECSig(sig)
	if err != nil {
		return false
	}
	return ecdsa.Verify(pub, msg, r, s)
}

package crypto

import (
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

package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"
)

func TestMarshal(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P224(), rand.Reader)
	if err != nil {
		t.Fatal("Error creating ECDSA key", err)
	}

	msg := make([]byte, 32)
	rand.Read(msg)

	r, s, err := ecdsa.Sign(rand.Reader, priv, msg)
	if err != nil {
		t.Fatal("Error signing", err)
	}

	marsh, err := MarshalECSig(r, s)
	if err != nil {
		t.Fatal("Error marshaling", err)
	}

	rp, sp, err := UnmarshalECSig(marsh)
	if err != nil {
		t.Fatal("Error unmarshaling", err)
	}

	if r.Cmp(rp) != 0 {
		t.Error("r != rp", r, rp)
	}
	if s.Cmp(sp) != 0 {
		t.Error("s != sp", s, sp)
	}

	if !ecdsa.Verify(&priv.PublicKey, msg, rp, sp) {
		t.Error("Invalid signature")
	}
}

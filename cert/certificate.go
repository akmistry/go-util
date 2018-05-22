package cert

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"time"
)

const (
	DefaultOrgName      = "Random Organisation"
	DefaultCommonName   = "domain.invalid"
	DefaultExpiryPeriod = 24 * time.Hour
)

var (
	serialNumberLimit = new(big.Int).Lsh(big.NewInt(1), 128)
)

type Options struct {
	OrgName    string
	CommonName string
	Expiry     time.Time
	PrivateKey crypto.PrivateKey
}

func GenerateCert(opts Options) (cert []byte, privKey crypto.PrivateKey) {
	if opts.OrgName == "" {
		opts.OrgName = DefaultOrgName
	}
	if opts.CommonName == "" {
		opts.CommonName = DefaultCommonName
	}
	if opts.Expiry.IsZero() {
		opts.Expiry = time.Now().Add(DefaultExpiryPeriod)
	}

	if opts.PrivateKey == nil {
		priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			panic(fmt.Sprintf("Failed to generate ECDSA key: %s", err))
		}
		opts.PrivateKey = priv
	}

	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		panic(fmt.Sprintf("Failed to generate serial number: %s", err))
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{opts.OrgName},
			CommonName:   opts.CommonName,
		},
		NotBefore: time.Now(),
		NotAfter:  opts.Expiry,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		BasicConstraintsValid: true,
	}

	var pubKey crypto.PublicKey
	switch k := opts.PrivateKey.(type) {
	case *rsa.PrivateKey:
		pubKey = &k.PublicKey
	case *ecdsa.PrivateKey:
		pubKey = &k.PublicKey
	default:
		panic("unsupported private key type")
	}
	derCert, err := x509.CreateCertificate(
		rand.Reader, &template, &template, pubKey, opts.PrivateKey)
	if err != nil {
		panic(fmt.Sprintf("Failed to generate certificate: %s", err))
	}

	return derCert, opts.PrivateKey
}

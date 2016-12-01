package crypto

import (
	"crypto/rand"
	"encoding/hex"
	"log"
)

func GenerateToken(length int) string {
	// Hex-encoding doubles the length, so generate 1 more byte than needed,
	// and then truncate the return value.
	buf := make([]byte, (length/2)+1)
	_, err := rand.Read(buf)
	if err != nil {
		log.Panicln("Unable to generate random token", err)
	}
	return hex.EncodeToString(buf)[:length]
}

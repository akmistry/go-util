package crypto

import (
	"crypto/rand"
	"encoding/hex"
	"log"
)

func GenerateToken(length int) string {
	buf := make([]byte, length)
        _, err := rand.Read(buf)
        if err != nil {
                log.Panicln("Unable to generate random token", err)
        }
        return hex.EncodeToString(buf)
}

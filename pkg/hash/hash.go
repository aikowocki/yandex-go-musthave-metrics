package hash

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

const HEADER = "HashSHA256"

func ComputeHMAC(key string, data []byte) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

func ValidateHMAC(key string, data []byte, expectedHex string) bool {
	expected, err := hex.DecodeString(expectedHex)
	if err != nil {
		return false
	}
	h := hmac.New(sha256.New, []byte(key))
	h.Write(data)
	return hmac.Equal(h.Sum(nil), expected)
}

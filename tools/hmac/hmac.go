package hmac

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"
	"strings"
)

func BuildPolyHmacSignature(secret string, timestamp string, method string, requestPath string, body *string) string {
	message := fmt.Sprintf("%s%s%s", timestamp, method, requestPath)
	if body != nil {
		message += *body
	}

	base64Secret, err := base64.URLEncoding.DecodeString(secret)
	if err != nil {
		log.Printf("secret decode  error: %v\n", err)
		base64Secret = []byte(secret)
	}

	h := hmac.New(sha256.New, base64Secret)
	h.Write([]byte(message))
	signature := h.Sum(nil)

	sig := base64.StdEncoding.EncodeToString(signature)

	return strings.ReplaceAll(strings.ReplaceAll(sig, "+", "-"), "/", "_")
}

// VerifyHmacSignature verifies an HMAC signature
func VerifyHmacSignature(secret string, timestamp string, method string, requestPath string, body *string, signature string) bool {
	expectedSig := BuildPolyHmacSignature(secret, timestamp, method, requestPath, body)
	return hmac.Equal([]byte(expectedSig), []byte(signature))
}

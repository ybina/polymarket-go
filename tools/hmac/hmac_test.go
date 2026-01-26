package hmac

import (
	"log"
	"strconv"
	"testing"
	"time"

	"github.com/ybina/polymarket-go/client/types"
)

func TestBuildAndVerifyHmac(t *testing.T) {
	apiCreds := &types.ApiKeyCreds{
		Key:        "",
		Secret:     "",
		Passphrase: "",
	}
	timestamp := time.Now().Unix()
	tsStr := strconv.FormatInt(timestamp, 10)
	method := "GET"
	requestPath := "/clob/api/v1/time"
	var body = "test body"

	signature := BuildPolyHmacSignature(
		apiCreds.Secret,
		tsStr,
		method,
		requestPath,
		&body,
	)

	if signature == "" {
		t.Fatal("signature should not be empty")
	}

	ok := VerifyHmacSignature(
		apiCreds.Secret,
		tsStr,
		method,
		requestPath,
		&body,
		signature,
	)

	if !ok {
		t.Fatal("expected HMAC verification to succeed")
	}

	badOk := VerifyHmacSignature(
		apiCreds.Secret,
		tsStr,
		"POST",
		requestPath,
		&body,
		signature,
	)

	if badOk {
		t.Fatal("expected HMAC verification to fail with wrong method")
	}
}

func Test_BuildPolyHmacSignature(t *testing.T) {
	secret := ""
	ts := ""
	method := "POST"
	path := "/order"
	body := ``
	signedHmacMsg := BuildPolyHmacSignature(secret, ts, method, path, &body)
	log.Printf("signedHmacMsg: %s\n", signedHmacMsg)
}

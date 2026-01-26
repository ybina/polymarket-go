package headers

import (
	"log"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ybina/polymarket-go/client/types"
)

func Test_CreateL2Headers(t *testing.T) {
	signer := common.HexToAddress("")
	ts := ""
	body := ``
	creds := types.ApiKeyCreds{
		Key:        "",
		Secret:     "",
		Passphrase: "",
	}
	headerArgs := types.L2HeaderArgs{
		Method:         "POST",
		RequestPath:    "/order",
		Body:           body,
		SerializedBody: body,
	}
	l2Header, err := CreateL2Headers(signer, &creds, &headerArgs, ts)
	if err != nil {
		t.Error(err)
		return
	}
	hStr, err := sonic.MarshalString(l2Header)
	if err != nil {
		t.Error(err)
		return
	}
	log.Printf("l2Header: %s\n", hStr)
}

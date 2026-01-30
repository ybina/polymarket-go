package bridge

import (
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ybina/polymarket-go/client/constants"
)

func TestBridgeClient_GetSupportedAssets(t *testing.T) {
	client, err := NewBridgeClient(&ClientConfig{
		Timeout:  20 * time.Second,
		ProxyUrl: "http://127.0.0.1:7890",
	})
	if err != nil {
		t.Fatal(err)
		return
	}
	assets, err := client.GetSupportedAssets()
	if err != nil {
		t.Fatal(err)
		return
	}
	assetsStr, err := sonic.MarshalString(assets)
	if err != nil {
		t.Fatal(err)
		return
	}
	t.Log(assetsStr)
}

func TestBridgeClient_CreateDepositAddresses(t *testing.T) {
	client, err := NewBridgeClient(&ClientConfig{
		Timeout:  20 * time.Second,
		ProxyUrl: "http://127.0.0.1:7890",
	})
	if err != nil {
		t.Fatal(err)
		return
	}
	safeAddr := common.HexToAddress("")
	res, err := client.CreateDepositAddress(safeAddr)
	if err != nil {
		t.Fatal(err)
		return
	}
	resStr, err := sonic.MarshalString(res)
	if err != nil {
		t.Fatal(err)
		return
	}
	t.Log(resStr)
}

func TestBridgeClient_GetAQuote(t *testing.T) {
	client, err := NewBridgeClient(&ClientConfig{
		Timeout:  20 * time.Second,
		ProxyUrl: "http://127.0.0.1:7890",
	})
	if err != nil {
		t.Fatal(err)
		return
	}
	req := QuoteRequest{
		FromAmountBaseUnit: "100000000000000000",
		FromChainID:        "56",
		FromTokenAddress:   "0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE",
		RecipientAddress:   constants.ZERO_ADDRESS.Hex(),
		ToChainID:          "137",
		ToTokenAddress:     "0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174",
	}
	res, err := client.GetAQuote(req)
	if err != nil {
		t.Fatal(err)
		return
	}
	resStr, err := sonic.MarshalString(res)
	if err != nil {
		t.Fatal(err)
		return
	}
	t.Log(resStr)

}

func TestBridgeClient_GetDepositStatus(t *testing.T) {
	client, err := NewBridgeClient(&ClientConfig{
		Timeout:  20 * time.Second,
		ProxyUrl: "http://127.0.0.1:7890",
	})
	if err != nil {
		t.Fatal(err)
		return
	}
	// from polymarket
	depositAddr := ""
	status, err := client.GetDepositStatus(depositAddr)
	if err != nil {
		t.Fatal(err)
		return
	}
	statusStr, err := sonic.MarshalString(status)

	if err != nil {
		t.Fatal(err)
		return
	}
	t.Log(statusStr)
}

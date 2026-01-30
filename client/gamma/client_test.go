package gamma

import (
	"log"
	"testing"

	"github.com/bytedance/sonic"
)

func TestGammaSDK_GetMarketByTokenId(t *testing.T) {
	tokenId := "25986405577356928223848081260299259484163711501323323218252464960086540660718"
	proxyUrl := "http://127.0.0.1:7890"
	client, err := NewGammaSDK(&proxyUrl)
	if err != nil {
		t.Error(err)
		return
	}
	query := &UpdatedMarketQuery{
		ClobTokenIDs: &tokenId,
	}
	res, err := client.GetMarkets(query)
	if err != nil {
		t.Fatal(err)
		return
	}
	resStr, err := sonic.MarshalString(res)
	if err != nil {
		t.Fatal(err)
		return
	}
	log.Println(resStr)
	log.Printf("conditionId:%v\n", res[0].ConditionID)
}

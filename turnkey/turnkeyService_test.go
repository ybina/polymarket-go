package turnkey

import (
	"log"
	"testing"

	"github.com/bytedance/sonic"
)

func TestClient_CreateAccount(t *testing.T) {
	turkeyConfig := Config{
		PubKey:       "",
		PrivateKey:   "",
		Organization: "",
		WalletName:   "",
	}
	client, err := NewTurnKeyService(turkeyConfig)
	if err != nil {
		t.Error(err)
		return
	}
	account, err := client.CreateAccount(7)
	if err != nil {
		t.Error(err)
		return
	}
	log.Printf("CreateAccount Account: %v\n", account)
}

func TestClient_GetAccount(t *testing.T) {
	turkeyConfig := Config{
		PubKey:       "",
		PrivateKey:   "",
		Organization: "",
		WalletName:   "",
	}
	client, err := NewTurnKeyService(turkeyConfig)
	if err != nil {
		t.Error(err)
		return
	}
	res, err := client.GetAccounts(7)
	if err != nil {
		t.Error(err)
		return
	}
	j, _ := sonic.MarshalString(res)
	log.Printf("GetAccounts: %s\n", j)

}

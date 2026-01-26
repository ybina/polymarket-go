package relayer

import (
	"fmt"
	"log"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
	"github.com/ybina/polymarket-go/client/relayer/builder"
	"github.com/ybina/polymarket-go/client/signer"
	"github.com/ybina/polymarket-go/client/types"
	"github.com/ybina/polymarket-go/tools/headers"
	"github.com/ybina/polymarket-go/turnkey"
)

func newRelayClient() (*RelayClient, error) {

	turkeyConfig := turnkey.Config{
		PubKey:       "",
		PrivateKey:   "",
		Organization: "",
		WalletName:   "",
	}
	signerConfig := signer.SignerConfig{
		SignerType:       signer.Turnkey,
		PrivateKeyConfig: nil,
		TurnkeyConfig:    &turkeyConfig,
		ChainID:          0,
	}
	s, err := signer.NewSigner(signerConfig)
	if err != nil {
		return nil, err
	}
	relayerUrl := "https://relayer-v2.polymarket.com"
	chainId := types.ChainPolygon

	builderConfig := headers.BuilderConfig{
		APIKey:     "",
		Secret:     "",
		Passphrase: "",
	}
	proxyUrl := ""
	relayClient, err := NewRelayClient(relayerUrl, chainId, s, &builderConfig, &proxyUrl)
	if err != nil {
		return nil, err
	}
	return relayClient, nil
}

func TestDeriveSafe(t *testing.T) {
	turnkeyAccount := common.HexToAddress("")

	relayClient, err := newRelayClient()
	if err != nil {
		t.Error(err)
		return
	}
	safe, res, err := relayClient.DeployWithTurnkey(turnkeyAccount)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("hash:%v \n", res.Hash)
	fmt.Printf("txhash:%v \n", res.TransactionHash)
	fmt.Printf("txId:%v \n", res.TransactionID)
	fmt.Printf("safe:%v \n", safe)
}

func TestRelayClient_ApproveForPolymarket(t *testing.T) {

	turnkeyAccount := common.HexToAddress("")

	relayClient, err := newRelayClient()
	if err != nil {
		t.Error(err)
		return
	}
	resp, err := relayClient.ApproveForPolymarketWithTurnkey(turnkeyAccount)
	if err != nil {
		t.Error(err)
		return
	}
	j, err := sonic.MarshalString(resp)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("resp:%s \n", j)
}

func TestRelayClient_GetNonce(t *testing.T) {
	turnkeyAccount := common.HexToAddress("")
	relayClient, err := newRelayClient()
	if err != nil {
		t.Error(err)
		return
	}
	safe := builder.Derive(turnkeyAccount, relayClient.ContractConfig.SafeFactory)
	log.Printf("safe:%v \n", safe.Hex())
	nonce, err := relayClient.GetNonce(safe, "SAFE")
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("nonce:%v \n", nonce)
}

func TestRelayClient_GetSafeNonceOnChain(t *testing.T) {
	turnkeyAccount := common.HexToAddress("")
	relayClient, err := newRelayClient()
	if err != nil {
		t.Error(err)
		return
	}
	safe := builder.Derive(turnkeyAccount, relayClient.ContractConfig.SafeFactory)
	log.Printf("safe:%v \n", safe.Hex())
	nonce, err := relayClient.GetSafeNonceOnChain(safe)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("nonce:%v \n", nonce)
}

func TestRelayClient_IsDeployed(t *testing.T) {
	turnkeyAccount := common.HexToAddress("")
	// turnkeyAccount := "0x48D20a994b3FF08026c7854b9f5B347c6116F03B"

	relayClient, err := newRelayClient()
	if err != nil {
		t.Error(err)
		return
	}
	safe := builder.Derive(turnkeyAccount, relayClient.ContractConfig.SafeFactory)
	log.Printf("safe:%v \n", safe)
	resp, err := relayClient.IsDeployed(safe)
	if err != nil {
		t.Error(err)
		return
	}
	j, _ := sonic.MarshalString(resp)
	fmt.Printf("resp:%s \n", j)

}

func TestRelayClient_CheckAllApprovals(t *testing.T) {
	turnkeyAccount := common.HexToAddress("")

	relayClient, err := newRelayClient()
	if err != nil {
		t.Error(err)
		return
	}
	safe := builder.Derive(turnkeyAccount, relayClient.ContractConfig.SafeFactory)
	log.Printf("safe:%v \n", safe.Hex())
	approved, usdcApprovals, tokenApprovals, err := relayClient.CheckAllApprovals(safe)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("approved:%v, usdcApprove:%v, tokenApprove:%v \n", approved, usdcApprovals, tokenApprovals)
}

func TestRelayClient_CheckUsdcApprovalForSpender(t *testing.T) {
	turnkeyAccount := common.HexToAddress("")

	relayClient, err := newRelayClient()
	if err != nil {
		t.Error(err)
		return
	}
	safe := builder.Derive(turnkeyAccount, relayClient.ContractConfig.SafeFactory)
	log.Printf("safe:%v \n", safe.Hex())
	op := common.HexToAddress("")
	ok, err := relayClient.CheckUsdcApprovalForSpender(safe, op)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("ok:%v \n", ok)
}

func TestRelayClient_CheckERC1155ApprovalForSpender(t *testing.T) {
	turnkeyAccount := common.HexToAddress("")

	relayClient, err := newRelayClient()
	if err != nil {
		t.Error(err)
		return
	}
	safe := builder.Derive(turnkeyAccount, relayClient.ContractConfig.SafeFactory)
	op := common.HexToAddress("")
	ok, err := relayClient.CheckERC1155ApprovalForSpender(safe, op)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("ok:%v \n", ok)
}

func TestRelayClient_GetTransaction(t *testing.T) {
	transaction := ""
	relayClient, err := newRelayClient()
	if err != nil {
		t.Error(err)
		return
	}
	getTransaction, err := relayClient.GetTransaction(transaction)
	if err != nil {
		t.Error(err)
		return
	}
	j, _ := sonic.MarshalString(getTransaction)
	fmt.Printf("getTransaction:%v \n", j)
}

func TestRelayClient_TransferUsdceFromSafeWithTurnkey(t *testing.T) {
	turnkeyAccount := common.HexToAddress("")
	target := common.HexToAddress("")
	amount := decimal.NewFromFloat(0.12)
	relayClient, err := newRelayClient()
	if err != nil {
		t.Error(err)
		return
	}
	tx, err := relayClient.TransferUsdceFromSafeWithTurnkey(turnkeyAccount, target, amount)
	if err != nil {
		t.Error(err)
		return
	}
	log.Printf("tx:%v \n", tx)
}

func TestRelayClient_GetNonceFromChain(t *testing.T) {
	turnkeyAccount := common.HexToAddress("")
	relayClient, err := newRelayClient()
	if err != nil {
		t.Error(err)
		return
	}
	safe := builder.Derive(turnkeyAccount, relayClient.ContractConfig.SafeFactory)
	log.Printf("safe:%v \n", safe.Hex())
	nonce, err := relayClient.GetSafeNonceOnChain(safe)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("nonce:%v \n", nonce)
}

func TestRelayClient_ApproveRequestContractForPolymarket(t *testing.T) {

	turnkeyAccount := common.HexToAddress("")
	contractName := "ctf_exchange"

	relayClient, err := newRelayClient()
	if err != nil {
		t.Error(err)
		return
	}
	resp, err := relayClient.ApproveRequestContractForPolymarketWithTurnkey(turnkeyAccount, contractName)
	if err != nil {
		t.Error(err)
		return
	}
	j, err := sonic.MarshalString(resp)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("resp:%s \n", j)
}

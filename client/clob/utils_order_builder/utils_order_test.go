package utils_order_builder

import (
	"fmt"
	"log"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ybina/polymarket-go/client/relayer/model/polyEip712"
	"github.com/ybina/polymarket-go/client/signer"
)

var eip712DomainTypeHash = crypto.Keccak256Hash([]byte(
	"EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)",
))

func TestOrder_OrderStructHash(t *testing.T) {

	tid := new(big.Int)
	tid, _ = tid.SetString("", 10)
	order := Order{
		Salt: big.NewInt(198180289),

		Maker: common.HexToAddress(""),

		Signer: common.HexToAddress(""),

		Taker: common.HexToAddress("0x0000000000000000000000000000000000000000"),

		TokenID: tid,

		MakerAmount: big.NewInt(1146000),

		TakerAmount: big.NewInt(6000000),

		Expiration: big.NewInt(0),

		Nonce: big.NewInt(0),

		FeeRateBps: big.NewInt(0),

		Side: uint8(0), // BUY

		SignatureType: uint8(0),
	}

	hash, err := order.OrderStructHash()
	if err != nil {
		t.Fatalf("OrderStructHash failed: %v", err)
	}
	t.Logf("OrderStructHash = %s", hash.Hex())
	name := "Polymarket CTF Exchange"
	version := "1"
	chainId := 137
	contract := "0xC5d563A36AE78145C45a50134d48A1215220f80a"

	domainSep := polyEip712.MakeDomain(&name, &version, &chainId, &contract, nil)
	hashDomain, err := domainSep.HashStruct()
	if err != nil {
		t.Fatalf("HashStruct failed: %v", err)
		return
	}
	fmt.Printf("domainSep = 0x%x\n", hashDomain)

	h := common.BytesToHash(hashDomain[:])

	digestHash := order.OrderEIP712Digest(h, hash)
	t.Logf("go digestHash = %s", digestHash.Hex())

	privateKey, err := crypto.HexToECDSA("")
	if err != nil {
		t.Errorf("failed to parse private key: %v", err)
		return
	}
	privateKeyConfig := signer.PrivateKeyClient{

		PrivateKey: privateKey,
		Address:    common.HexToAddress(""),
	}
	signerConfig := signer.SignerConfig{
		SignerType:       signer.PrivateKey,
		ChainID:          137,
		PrivateKeyConfig: &privateKeyConfig,
	}
	signerHandler, err := signer.NewSigner(signerConfig)
	if err != nil {
		log.Fatal(err)
	}
	signHash, err := signerHandler.SignHash(digestHash.Hex())
	if err != nil {
		t.Errorf("failed to sign hash: %v", err)
		return
	}
	log.Printf("signed Msg: %v\n", signHash)

}

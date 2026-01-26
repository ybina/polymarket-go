package clob

import (
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/shopspring/decimal"
	"github.com/ybina/polymarket-go/client/clob/clob_types"
	config2 "github.com/ybina/polymarket-go/client/config"
	"github.com/ybina/polymarket-go/client/constants"
	"github.com/ybina/polymarket-go/client/relayer/builder"
	"github.com/ybina/polymarket-go/client/signer"
	"github.com/ybina/polymarket-go/client/types"
	"github.com/ybina/polymarket-go/tools/headers"
	"github.com/ybina/polymarket-go/turnkey"
)

func initClobClientWithTurnkey(turnkeyAccount common.Address) (client *ClobClient, safeAddr *common.Address, err error) {
	builderConfig := &headers.BuilderConfig{
		APIKey:     "",
		Secret:     "",
		Passphrase: "",
	}
	apiCreds := &types.ApiKeyCreds{}

	turnkeyConfig := turnkey.Config{
		PubKey:       "",
		PrivateKey:   "",
		Organization: "",
		WalletName:   "",
	}
	signerConfig := signer.SignerConfig{
		SignerType:    signer.Turnkey,
		ChainID:       137,
		TurnkeyConfig: &turnkeyConfig,
	}
	signerHandler, err := signer.NewSigner(signerConfig)
	if err != nil {
		log.Fatal(err)
	}
	config := &ClientConfig{
		Host:          "https://clob.polymarket.com",
		ChainID:       types.ChainPolygon,
		Signer:        signerHandler,
		APIKey:        apiCreds,
		UseServerTime: true,
		Timeout:       30 * time.Second,
		ProxyUrl:      "",
		BuilderConfig: builderConfig,
	}

	clobClient, err := NewClobClient(config)
	if err != nil {
		return nil, nil, err
	}
	fmt.Println("CLOB client created successfully")

	serverTime, err := clobClient.GetServerTime()
	if err != nil {
		return nil, nil, err
	} else {
		fmt.Printf("Server time: %d\n", serverTime)
	}

	fmt.Println("Creating API key...")
	option := clob_types.ClobOption{
		TurnkeyAccount: turnkeyAccount,
	}

	contractConfig, err := config2.GetContractConfig(types.Chain(signerHandler.ChainID()))
	if err != nil {
		return nil, nil, err
	}
	option.SafeAccount = builder.Derive(option.TurnkeyAccount, contractConfig.SafeFactory)
	log.Printf("safe: %v\n", option.SafeAccount.Hex())
	apiKey, err := clobClient.CreateApiKey(nil, option)
	if err != nil {
		log.Printf("Failed to create API key: %v", err)
		log.Printf("Start derive API creds ... \n")
		apiCreds, err = clobClient.DeriveApiKey(nil, option)
		if err != nil {
			log.Printf("Failed to derive API creds: %v", err)
			return
		}
		fmt.Printf("API Key derive successfully:\n")
		fmt.Printf("API  Key: %s\n", apiCreds.Key)
		fmt.Printf("API  secret: %s\n", apiCreds.Secret)
		fmt.Printf("API passphrase: %s\n", apiCreds.Passphrase)

		config.APIKey = apiCreds
		clobClient, err = NewClobClient(config)
		if err != nil {
			return nil, nil, err
		}
	} else {
		fmt.Printf("API Key created successfully:\n")
		fmt.Printf("API  Key: %s\n", apiKey.Key)
		fmt.Printf("API  secret: %s\n", apiKey.Secret)
		fmt.Printf("API passphrase: %s\n", apiKey.Passphrase)

		apiCreds = apiKey
		config.APIKey = apiCreds

		clobClient, err = NewClobClient(config)
		if err != nil {
			return nil, nil, err
		}
	}
	return clobClient, &option.SafeAccount, nil
}

func TestClobClient_GetPrice(t *testing.T) {
	tokenId := ""
	config := &ClientConfig{
		Host:          "https://clob.polymarket.com",
		ChainID:       types.ChainPolygon,
		Signer:        nil,
		APIKey:        nil,
		UseServerTime: true,
		Timeout:       30 * time.Second,
		ProxyUrl:      "",
	}

	clobClient, err := NewClobClient(config)
	if err != nil {
		t.Fatal(err)
	}
	res, err := clobClient.GetPrice(tokenId, types.SideBuy)
	if err != nil {
		t.Fatal(err)
	}
	log.Printf("res: %v", res)

}

func Test_CreateClobClient(t *testing.T) {
	pubKey := ""
	privateKey := ""
	builderConfig := &headers.BuilderConfig{
		APIKey:     "",
		Secret:     "",
		Passphrase: "",
	}

	apiCreds := &types.ApiKeyCreds{}
	priKey, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		log.Fatal(err)
	}
	privateKeyConfig := signer.PrivateKeyClient{
		Address:    common.HexToAddress(pubKey),
		PrivateKey: priKey,
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
	config := &ClientConfig{
		Host:          "https://clob.polymarket.com",
		ChainID:       types.ChainPolygon, // 137 for Polygon
		Signer:        signerHandler,
		APIKey:        apiCreds,
		UseServerTime: true,
		Timeout:       30 * time.Second,
		ProxyUrl:      "",
		BuilderConfig: builderConfig,
	}

	clobClient, err := NewClobClient(config)
	if err != nil {
		t.Errorf("failed to create clobClient: %v", err)
	}
	fmt.Println("CLOB client created successfully")

	serverTime, err := clobClient.GetServerTime()
	if err != nil {
		t.Errorf("failed to get server time: %v", err)
	} else {
		fmt.Printf("Server time: %d\n", serverTime)
	}

	fmt.Println("Creating API key...")
	option := clob_types.ClobOption{}

	apiKey, err := clobClient.CreateApiKey(nil, option)
	if err != nil {
		log.Printf("Failed to create API key: %v", err)
		fmt.Println("Note: This might fail if you already have an API key")
		log.Printf("Start derive API creds ... \n")
		apiCreds, err = clobClient.DeriveApiKey(nil, option)
		if err != nil {
			log.Printf("Failed to derive API creds: %v", err)
		}
		log.Printf("derive API creds: %v\n", apiCreds)
		config.APIKey = apiCreds
		clobClient, err = NewClobClient(config)
		if err != nil {
			t.Errorf("failed to create clobClient: %v", err)
		}
	} else {
		fmt.Printf("API Key created successfully:\n")
		fmt.Printf("  Key: %s\n", apiKey.Key)
		fmt.Printf("  (Secret and passphrase are sensitive)\n")

		apiCreds = apiKey
		config.APIKey = apiCreds

		clobClient, err = NewClobClient(config)
		if err != nil {
			t.Errorf("failed to create clobClient: %v", err)
		}
	}
	funder := common.HexToAddress(signerHandler.Address())

	if apiCreds != nil {

		apiKeys, err := clobClient.GetApiKeys(funder)
		if err != nil {
			t.Errorf("failed to get api keys: %v", err)
		} else {
			fmt.Printf("Found %d API keys\n", len(apiKeys.APIKeys))
		}

		banStatus, err := clobClient.GetClosedOnlyMode(funder)
		if err != nil {
			t.Errorf("failed to get closed only mode: %v", err)
		} else {
			fmt.Printf("Closed only mode: %v\n", banStatus.ClosedOnly)
		}

		trades, err := clobClient.GetTrades(funder, nil, true, "")
		if err != nil {
			t.Errorf("failed to get trades: %v", err)
		} else {
			fmt.Printf("Found %d trades\n", len(trades))
		}
	}

}

func Test_CreateClobClientWithTurnkey(t *testing.T) {
	builderConfig := &headers.BuilderConfig{
		APIKey:     "",
		Secret:     "",
		Passphrase: "",
	}

	apiCreds := &types.ApiKeyCreds{}
	turnkeyConfig := turnkey.Config{
		PubKey:       "",
		PrivateKey:   "",
		Organization: "",
		WalletName:   "",
	}
	signerConfig := signer.SignerConfig{
		SignerType:    signer.Turnkey,
		ChainID:       137,
		TurnkeyConfig: &turnkeyConfig,
	}
	signerHandler, err := signer.NewSigner(signerConfig)
	if err != nil {
		log.Fatal(err)
	}
	config := &ClientConfig{
		Host:          "https://clob.polymarket.com",
		ChainID:       types.ChainPolygon, // 137 for Polygon
		Signer:        signerHandler,
		UseServerTime: true,
		Timeout:       30 * time.Second,
		ProxyUrl:      "",
		BuilderConfig: builderConfig,
	}

	clobClient, err := NewClobClient(config)
	if err != nil {
		t.Errorf("failed to create clobClient: %v", err)
	}
	fmt.Println("CLOB client created successfully")

	serverTime, err := clobClient.GetServerTime()
	if err != nil {
		t.Errorf("failed to get server time: %v", err)
	} else {
		fmt.Printf("Server time: %d\n", serverTime)
	}

	fmt.Println("Creating API key...")
	option := clob_types.ClobOption{
		TurnkeyAccount: common.HexToAddress(""),
	}
	contractConfig, err := config2.GetContractConfig(types.Chain(signerHandler.ChainID()))
	if err != nil {
		t.Errorf("failed to get contract config: %v", err)
	}
	option.SafeAccount = builder.Derive(option.TurnkeyAccount, contractConfig.SafeFactory)
	log.Printf("safe: %v\n", option.SafeAccount.Hex())
	apiKey, err := clobClient.CreateApiKey(nil, option)
	if err != nil {
		log.Printf("Failed to create API key: %v", err)
		fmt.Println("Note: This might fail if you already have an API key")
		log.Printf("Start derive API creds ... \n")
		apiCreds, err = clobClient.DeriveApiKey(nil, option)
		if err != nil {
			log.Printf("Failed to derive API creds: %v", err)
			return
		}
		fmt.Printf("api  key: %s\n", apiCreds.Key)
		fmt.Printf("api secret: %s\n", apiCreds.Secret)
		fmt.Printf("api passphrase:%s\n", apiCreds.Passphrase)
		config.APIKey = apiCreds
		clobClient, err = NewClobClient(config)
		if err != nil {
			t.Errorf("failed to create clobClient: %v", err)
		}
	} else {
		fmt.Printf("API Key created successfully:\n")
		fmt.Printf("api  key: %s\n", apiKey.Key)
		fmt.Printf("api secret: %s\n", apiKey.Secret)
		fmt.Printf("api passphrase:%s\n", apiKey.Passphrase)

		apiCreds = apiKey
		config.APIKey = apiCreds

		clobClient, err = NewClobClient(config)
		if err != nil {
			t.Errorf("failed to create clobClient: %v", err)
		}
	}
	addr := option.TurnkeyAccount

	if config.APIKey != nil {
		apiKeys, err := clobClient.GetApiKeys(addr)
		if err != nil {
			t.Errorf("failed to get api keys: %v", err)
			return
		} else {
			fmt.Printf("Found %d API keys\n", len(apiKeys.APIKeys))
		}

		banStatus, err := clobClient.GetClosedOnlyMode(addr)
		if err != nil {
			t.Errorf("failed to get closed only mode: %v", err)
			return
		} else {
			fmt.Printf("Closed only mode: %v\n", banStatus.ClosedOnly)
		}

		trades, err := clobClient.GetTrades(addr, nil, true, "")
		if err != nil {
			t.Errorf("failed to get trades: %v", err)
		} else {
			fmt.Printf("Found %d trades\n", len(trades))
		}
	}
}

func Test_TurnkeyCreateOrder(t *testing.T) {
	turnkeyAccount := common.HexToAddress("")
	clobClient, safeAddr, err := initClobClientWithTurnkey(turnkeyAccount)
	if err != nil {
		t.Errorf("failed to init clobClient: %v", err)
		return
	}
	if safeAddr == nil || *safeAddr == constants.ZERO_ADDRESS {
		t.Errorf("invalid safeAddr")
		return
	}
	price, err := decimal.NewFromString("0.15")
	size, err := decimal.NewFromString("8")
	args := clob_types.OrderArgs{
		TokenID:    "",
		Price:      price,
		Size:       size,
		Side:       "BUY",
		FeeRateBps: 0,
		Nonce:      0,
		Expiration: 0,
		Taker:      constants.ZERO_ADDRESS,
	}
	orderOption := clob_types.PartialCreateOrderOptions{
		OrderType:      types.OrderTypeGTC,
		TurnkeyAccount: turnkeyAccount,
		SafeAccount:    *safeAddr,
	}
	resp, err := clobClient.CreateAndPostOrder(args, orderOption)
	if err != nil {
		t.Errorf("failed to create order: %v", err)
		return
	}
	str, err := sonic.MarshalString(resp)
	if err != nil {
		t.Errorf("failed to marshal response: %v", err)
		return
	}
	fmt.Println(str)
}

func Test_PrivateKeyCreateOrder(t *testing.T) {

	privateKey, err := crypto.HexToECDSA("")
	if err != nil {
		t.Errorf("%v", err)
		return
	}

	builderConfig := &headers.BuilderConfig{
		APIKey:     "",
		Secret:     "",
		Passphrase: "",
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
	config := &ClientConfig{
		Host:          "https://clob.polymarket.com",
		ChainID:       types.ChainPolygon, // 137 for Polygon
		Signer:        signerHandler,
		UseServerTime: true,
		Timeout:       30 * time.Second,
		ProxyUrl:      "",
		BuilderConfig: builderConfig,
	}

	clobClient, err := NewClobClient(config)
	if err != nil {
		t.Errorf("failed to create clobClient: %v", err)
	}
	fmt.Println("CLOB client created successfully")

	serverTime, err := clobClient.GetServerTime()
	if err != nil {
		t.Errorf("failed to get server time: %v", err)
	} else {
		fmt.Printf("Server time: %d\n", serverTime)
	}

	option := clob_types.ClobOption{}
	if err != nil {
		t.Errorf("failed to get contract config: %v", err)
	}

	apiKey, err := clobClient.CreateApiKey(nil, option)
	if err != nil {
		log.Printf("Failed to create API key: %v", err)
		fmt.Println("Note: This might fail if you already have an API key")
		log.Printf("Start derive API creds ... \n")
		apiCreds, err := clobClient.DeriveApiKey(nil, option)
		if err != nil {
			log.Printf("Failed to derive API creds: %v", err)
			return
		}
		log.Printf("derive API creds: %v\n", apiCreds)
		config.APIKey = apiCreds
		clobClient, err = NewClobClient(config)
		if err != nil {
			t.Errorf("failed to create clobClient: %v", err)
		}
	} else {
		fmt.Printf("API Key created successfully:\n")
		fmt.Printf("API Key for private key: %s\n", apiKey)

		apiCreds := apiKey
		config.APIKey = apiCreds

		clobClient, err = NewClobClient(config)
		if err != nil {
			t.Errorf("failed to create clobClient: %v", err)
		}
	}
	price, err := decimal.NewFromString("0.191")
	size, err := decimal.NewFromString("6")
	args := clob_types.OrderArgs{
		TokenID:    "",
		Price:      price,
		Size:       size,
		Side:       "BUY",
		FeeRateBps: 0,
		Nonce:      0,
		Expiration: 0,
		Taker:      constants.ZERO_ADDRESS,
	}
	orderOption := clob_types.PartialCreateOrderOptions{
		OrderType:      types.OrderTypeGTC,
		TurnkeyAccount: option.TurnkeyAccount,
		SafeAccount:    option.SafeAccount,
	}
	resp, err := clobClient.CreateAndPostOrder(args, orderOption)
	if err != nil {
		t.Errorf("failed to create order: %v", err)
		return
	}
	str, err := sonic.MarshalString(resp)
	if err != nil {
		t.Errorf("failed to marshal response: %v", err)
		return
	}
	fmt.Println(str)
}

func Test_TurnkeyCreateMarketOrder(t *testing.T) {
	turnkeyAccount := common.HexToAddress("")
	clobClient, safe, err := initClobClientWithTurnkey(turnkeyAccount)
	if err != nil {
		t.Errorf("failed to create clobClient: %v", err)
		return
	}
	price, err := decimal.NewFromString("0.33")
	amount := decimal.NewFromFloat(3.235293)
	args := clob_types.MarketOrderArgs{
		TokenID:    "",
		Price:      price,
		Amount:     amount,
		Side:       "SELL",
		FeeRateBps: 0,
		Nonce:      0,

		Taker: constants.ZERO_ADDRESS,
	}
	orderOption := clob_types.PartialCreateOrderOptions{
		OrderType:      types.OrderTypeFOK,
		TurnkeyAccount: turnkeyAccount,
		SafeAccount:    *safe,
	}
	resp, err := clobClient.CreateAndPostMarketOrder(args, orderOption)
	if err != nil {
		t.Errorf("failed to create order: %v", err)
		return
	}
	str, err := sonic.MarshalString(resp)
	if err != nil {
		t.Errorf("failed to marshal response: %v", err)
		return
	}

	fmt.Println(str)
}

func TestClobClient_GetTrades(t *testing.T) {
	turnkeyAccount := common.HexToAddress("")
	clobClient, _, err := initClobClientWithTurnkey(turnkeyAccount)
	if err != nil {
		t.Errorf("failed to create clobClient: %v", err)
		return
	}
	trades, err := clobClient.GetTrades(turnkeyAccount, nil, true, "")
	if err != nil {
		t.Errorf("failed to get trades: %v", err)
	} else {
		fmt.Printf("Found %d trades\n", len(trades))
		tradesStr, err := sonic.MarshalString(trades)
		if err != nil {
			t.Errorf("failed to marshal trades: %v", err)
			return
		}
		fmt.Printf("trades: %s\n", tradesStr)
	}
}

func TestClobClient_GetOrder(t *testing.T) {
	turnkeyAccount := common.HexToAddress("")
	orderId := ""
	clobClient, _, err := initClobClientWithTurnkey(turnkeyAccount)
	if err != nil {
		t.Errorf("failed to create clobClient: %v", err)
		return
	}
	order, err := clobClient.GetOrder(turnkeyAccount, orderId)
	if err != nil {
		t.Errorf("failed to get order: %v", err)
		return
	}
	orderStr, err := sonic.MarshalString(order)
	if err != nil {
		t.Errorf("failed to marshal order: %v", err)
		return
	}
	fmt.Printf("Order: %v\n", orderStr)
}

func TestClobClient_CancelOrder(t *testing.T) {
	turnkeyAccount := common.HexToAddress("")
	orderId := ""
	clobClient, _, err := initClobClientWithTurnkey(turnkeyAccount)
	if err != nil {
		t.Errorf("failed to create clobClient: %v", err)
		return
	}
	res, err := clobClient.CancelOrder(orderId, turnkeyAccount)
	if err != nil {
		t.Errorf("failed to cancel order: %v", err)
		return
	}
	resStr, err := sonic.MarshalString(res)
	if err != nil {
		t.Errorf("failed to marshal order: %v", err)
		return
	}
	fmt.Printf("Order resp: %v\n", resStr)
}

func TestGetPrice(t *testing.T) {
	turnkeyAccount := common.HexToAddress("")
	tokenId := ""
	side := types.SideBuy
	clobClient, _, err := initClobClientWithTurnkey(turnkeyAccount)
	if err != nil {
		t.Errorf("failed to create clobClient: %v", err)
		return
	}
	res, err := clobClient.GetPrice(tokenId, side)
	if err != nil {
		t.Errorf("failed to get price: %v", err)
		return
	}
	log.Printf("Price: %v\n", res.String())
}

func TestClobClient_GetTickSize(t *testing.T) {
	tokenId := ""
	turnkeyAccount := common.HexToAddress("")
	clobClient, _, err := initClobClientWithTurnkey(turnkeyAccount)
	if err != nil {
		t.Errorf("failed to create clobClient: %v", err)
		return
	}
	tickSize, err := clobClient.GetTickSize(tokenId)
	if err != nil {
		t.Errorf("failed to get tick size: %v", err)
		return
	}
	log.Printf("Tick size: %v\n", tickSize)
}

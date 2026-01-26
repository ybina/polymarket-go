# polymarket-go

A Go SDK for **Polymarket** (community/third‑party implementation). It covers **CLOB**, **Relayer**, **Data API**, **Gamma API**, and **WebSocket** market data—and additionally supports **Turnkey** for wallet/user management and signing trades.

> **Disclaimer:** This is not an official Polymarket library. APIs and request/response shapes may change. Use at your own risk and add proper monitoring, retries, and safeguards in production.  
> **Security:** Do **not** commit real private keys, Turnkey credentials, or API keys into the repo.

---

## Features

- **CLOB (REST)**: orderbook, markets, prices, trades, order management, rewards/notifications, etc. (`client/clob`)
- **Trading & Signing**
  - EIP‑712 signing (CLOB **L1** headers)
  - HMAC signing (CLOB **L2** headers)
  - Optional Builder headers (builder/relayer flows)
  - Order builder for common order types
- **Relayer**
  - Nonce, submit/query transactions, Safe deployment status (`client/relayer`)
  - Helpers for Safe transaction construction/signing (Turnkey-friendly)
- **Data API / Gamma API**
  - Typed HTTP clients and models (`client/data`, `client/gamma`)
- **WebSocket**
  - Connects to `wss://ws-subscriptions-clob.polymarket.com`
  - Auto-reconnect, ping/pong, subscribe/unsubscribe, callback dispatch (`client/ws`)
- **Turnkey Integration**
  - Create/reuse a master wallet
  - Create sub-accounts (BIP32 paths)
  - Sign payloads via Turnkey (`turnkey`, `client/signer`)

---

## Project Layout

- `client/clob` – CLOB REST client + order placement/cancel/query + L1/L2 header composition
- `client/ws` – WebSocket client for market data (books/prices)
- `client/relayer` – Relayer client (nonce/submit/tx/deployed) + Safe helpers
- `client/data` – `data-api.polymarket.com` client
- `client/gamma` – `gamma-api.polymarket.com` client
- `client/signer` – unified signer (PrivateKey / Turnkey)
- `turnkey` – Turnkey wallet management + signing
- `tools/*` – EIP712 / HMAC / headers / general utilities

---

## Requirements

- Go: `go.mod` declares `go 1.25.6` (if you use a lower Go version locally, adjust and run tests)
- Key deps (not exhaustive):
  - `github.com/ethereum/go-ethereum`
  - `github.com/gorilla/websocket`
  - `github.com/shopspring/decimal`
  - `github.com/tkhq/go-sdk` (Turnkey)

---

## Install

```bash
go get github.com/ybina/polymarket-go
```

---

## Quick Start

Below are minimal examples for common flows using **PrivateKey** and **Turnkey** signers.

### 1) PrivateKey signer + CLOB client + orderbook

```go
package main

import (
	"log"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ybina/polymarket-go/client/clob"
	"github.com/ybina/polymarket-go/client/signer"
	"github.com/ybina/polymarket-go/client/types"
	"github.com/ybina/polymarket-go/tools/headers"
)

func main() {
	// 1) Signer (PrivateKey)
	pk, err := crypto.HexToECDSA("YOUR_PRIVATE_KEY_HEX_WITHOUT_0x")
	if err != nil {
		log.Fatal(err)
	}

	sk, err := signer.NewSigner(signer.SignerConfig{
		SignerType: signer.PrivateKey,
		ChainID:    137,
		PrivateKeyConfig: &signer.PrivateKeyClient{
			PrivateKey: pk,
			Address:    common.HexToAddress("0xYourAddress"),
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	// 2) Optional: builder config (if you use builder headers / builder trades)
	builderCfg := &headers.BuilderConfig{
		APIKey:     "YOUR_BUILDER_API_KEY",
		Secret:     "YOUR_BUILDER_SECRET",
		Passphrase: "YOUR_BUILDER_PASSPHRASE",
	}

	// 3) Create CLOB client
	c, err := clob.NewClobClient(&clob.ClientConfig{
		Host:          "https://clob.polymarket.com",
		ChainID:       types.ChainPolygon,
		Signer:        sk,
		UseServerTime: true,
		Timeout:       30 * time.Second,
		ProxyUrl:      "", // e.g. "http://127.0.0.1:7890"
		BuilderConfig: builderCfg,
	})
	if err != nil {
		log.Fatal(err)
	}

	// 4) Read orderbook
	ob, err := c.GetOrderBook("TOKEN_ID")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("asks=%d bids=%d", len(ob.Asks), len(ob.Bids))
}
```

### 2) Derive API key (L2) and place an order

`CreateApiKey` may fail if the key already exists; `DeriveApiKey` lets you reuse credentials.

```go
package main

import (
	"log"

	"github.com/shopspring/decimal"
	"github.com/ybina/polymarket-go/client/clob"
	"github.com/ybina/polymarket-go/client/clob/clob_types"
	"github.com/ybina/polymarket-go/client/types"
)

func main() {
	// assume clobClient was initialized (see previous example)
	var clobClient *clob.ClobClient

	// 1) Get API creds
	apiCreds, err := clobClient.DeriveApiKey(nil, clob_types.ClobOption{})
	if err != nil {
		log.Fatal(err)
	}
	_ = apiCreds

	// 2) Place a limit order
	price := decimal.RequireFromString("0.15")
	size := decimal.RequireFromString("8")

	resp, err := clobClient.CreateAndPostOrder(
		clob_types.OrderArgs{
			TokenID:    "TOKEN_ID",
			Price:      price,
			Size:       size,
			Side:       "BUY",
			FeeRateBps: 0,
			Nonce:      0,
			Expiration: 0,
		},
		clob_types.PartialCreateOrderOptions{
			OrderType: types.OrderTypeGTC,
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("order id=%s status=%s", resp.OrderID, resp.Status)
}
```

> Before trading with real funds, validate market status, min tick size, fee rates, balances, and allowances.

### 3) Turnkey: create/reuse master wallet + create a sub-account + sign

```go
package main

import (
	"log"

	"github.com/ybina/polymarket-go/turnkey"
)

func main() {
	tk, err := turnkey.NewTurnKeyService(turnkey.Config{
		PubKey:       "YOUR_TURNKEY_PUBLIC_KEY",
		PrivateKey:   "YOUR_TURNKEY_PRIVATE_KEY",
		Organization: "YOUR_ORG_ID",
		WalletName:   "your-master-wallet-name",
	})
	if err != nil {
		log.Fatal(err)
	}

	// Create the N-th address (m/44'/60'/0'/0/N)
	addr, err := tk.CreateAccount(1)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("new addr:", addr)
}
```

Use a Turnkey signer in the SDK:

```go
sk, err := signer.NewSigner(signer.SignerConfig{
	SignerType: signer.Turnkey,
	ChainID:    137,
	TurnkeyConfig: &turnkey.Config{
		PubKey:       "YOUR_TURNKEY_PUBLIC_KEY",
		PrivateKey:   "YOUR_TURNKEY_PRIVATE_KEY",
		Organization: "YOUR_ORG_ID",
		WalletName:   "your-master-wallet-name",
	},
})
```

### 3) Turnkey: deploy SAFE wallet and approve for polymarket

```go
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ybina/polymarket-go/client/relayer"
	"github.com/ybina/polymarket-go/client/signer"
	"github.com/ybina/polymarket-go/client/types"
	"github.com/ybina/polymarket-go/tools/headers"
	"github.com/ybina/polymarket-go/turnkey"
)

func main() {
	tkCfg := turnkey.Config{
		PubKey:       "YOUR_TURNKEY_PUBLIC_KEY",
		PrivateKey:   "YOUR_TURNKEY_PRIVATE_KEY",
		Organization: "YOUR_ORG_ID",
		WalletName:   "YOUR_MASTER_WALLET_NAME",
	}

	sk, err := signer.NewSigner(signer.SignerConfig{
		SignerType:    signer.Turnkey,
		TurnkeyConfig: &tkCfg,
		ChainID:       137, 
	if err != nil {
		log.Fatal(err)
	}
	
	builderCfg := headers.BuilderConfig{
		APIKey:     "YOUR_BUILDER_API_KEY",
		Secret:     "YOUR_BUILDER_SECRET",
		Passphrase: "YOUR_BUILDER_PASSPHRASE",
	}
	
	relayerURL := "https://relayer-v2.polymarket.com"
	chain := types.ChainPolygon
	proxyURL := "" // option "http://127.0.0.1:7890"

	rc, err := relayer.NewRelayClient(relayerURL, chain, sk, &builderCfg, &proxyURL)
	if err != nil {
		log.Fatal(err)
	}
	
	turnkeyAccount := common.HexToAddress("0xYOUR_TURNKEY_ACCOUNT_ADDRESS")
	
	safeAddr, deployResp, err := rc.DeployWithTurnkey(turnkeyAccount)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("[DEPLOY] safe=%s txID=%s txHash=%s hash=%s\n",
		safeAddr.Hex(),
		deployResp.TransactionID,
		deployResp.TransactionHash,
		deployResp.Hash,
	)
	
	deadline := time.Now().Add(90 * time.Second)
	for {
		ok, err := rc.IsDeployed(safeAddr)
		if err != nil {
			log.Fatal(err)
		}
		if ok {
			break
		}
		if time.Now().After(deadline) {
			log.Fatalf("safe not deployed within timeout: %s", safeAddr.Hex())
		}
		time.Sleep(3 * time.Second)
	}
	fmt.Printf("[DEPLOY] confirmed deployed: %s\n", safeAddr.Hex())
	approveResp, err := rc.ApproveForPolymarketWithTurnkey(turnkeyAccount)
	if err != nil {
		log.Fatal(err)
	}
	if approveResp == nil {
		fmt.Println("[APPROVE] already approved; nothing to do")
		return
	}

	fmt.Printf("[APPROVE] txID=%s txHash=%s hash=%s\n",
		approveResp.TransactionID,
		approveResp.TransactionHash,
		approveResp.Hash,
	)
}
```

### 4) WebSocket: subscribe to market channel

```go
wsClient := ws.NewWebSocketClient(clobClient, &ws.WebSocketClientOptions{
	AutoReconnect:  true,
	ReconnectDelay: 5 * time.Second,
	ProxyUrl:       "", // optional proxy
})

wsClient.On(&ws.WebSocketCallbacks{
	OnBook: func(msg *types.BookMessage) {
		// handle orderbook updates
	},
	OnError: func(err error) { log.Println("ws err:", err) },
})

if err := wsClient.Connect(); err != nil {
	log.Fatal(err)
}
_ = wsClient.Subscribe([]string{"TOKEN_ID"})
```

---

## Testing

There are `*_test.go` files (e.g. `client/clob/clob_client_test.go`, `client/gamma/client_test.go`, `turnkey/turnkeyService_test.go`).  
Some tests are intended for local debugging; make sure you **do not** hard-code real credentials.

```bash
go test ./...
```

---

## Notes / Suggested Improvements

Your SDK is already quite complete. If you want to make it more “production-ready”, consider:

- unified HTTP client: retries, backoff, rate limiting, structured logging, tracing
- consistent error types (endpoint/status/body parsing)
- more end-to-end examples for Turnkey + Safe relayer transaction flows
- richer `godoc` examples (`Example*`) for key entry points

---

## License

MIT (see `LICENSE`)

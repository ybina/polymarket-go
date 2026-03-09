# polymarket-go

A Go SDK for **Polymarket** (community / third-party implementation).
Covers **CLOB**, **Relayer**, **Data API**, **Gamma API**, **Bridge**, and **WebSocket** market data—with optional **Turnkey** integration for wallet management and signing.

> **Disclaimer:** This is not an official Polymarket library. APIs and response shapes may change without notice. Use at your own risk and add proper monitoring, retries, and safeguards in production.

> **Security:** Never commit real private keys, Turnkey credentials, or API keys to the repository.

> **Note:** Only gas-less transactions are supported; you must derive a Gnosis Safe wallet before trading.

> **Tip:** Use your own Polygon RPC URL in the relay client for better reliability.

If this project is helpful, feel free to give it a star ⭐

---

## Features

| Category | Description |
|---|---|
| **CLOB (REST)** | Orderbook, markets, prices, trades, order management, rewards/notifications |
| **Trading & Signing** | EIP-712 (L1), HMAC (L2), optional Builder headers, order builder for all common order types |
| **Relayer** | Nonce, submit/query transactions, Safe deployment status, Safe tx construction/signing |
| **Data API** | Positions, closed positions, trades, activity, top holders, open interest, volume |
| **Gamma API** | Markets, events, search, tags, series, comments |
| **WebSocket** | `wss://ws-subscriptions-clob.polymarket.com` — auto-reconnect, ping/pong, subscribe/unsubscribe |
| **Bridge** | Create deposit addresses, get supported assets, check deposit status, get quotes |
| **Turnkey** | Create/reuse master wallets, derive sub-accounts (BIP-32), sign via Turnkey API |
| **Proxy** | Pass a proxy URL to any client; set to `""` to disable |

---

## Project Layout

```
client/
  clob/     – CLOB REST client: order placement, cancel, query, L1/L2 headers
  ws/       – WebSocket client for real-time book & price data
  relayer/  – Relayer client: nonce, submit tx, Safe helpers
  data/     – data-api.polymarket.com client
  gamma/    – gamma-api.polymarket.com client
  signer/   – Unified signer (PrivateKey / Turnkey)
  bridge/   – Multi-chain asset bridge client
turnkey/    – Turnkey wallet management & signing
tools/      – EIP-712, HMAC, header utilities
```

---

## Requirements

- **Go:** `go 1.25.6` (declared in `go.mod`; adjust if using an older local toolchain)
- **Key dependencies:**
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

The examples below cover the two main signing flows—**PrivateKey** and **Turnkey**—plus Bridge and WebSocket.

---

### Flow A — PrivateKey Signer

#### Step 1: Initialize the CLOB Client

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
    // --- Signer ---
    pk, err := crypto.HexToECDSA("YOUR_PRIVATE_KEY_HEX_WITHOUT_0x")
    if err != nil {
        log.Fatal(err)
    }
    sk, err := signer.NewSigner(signer.SignerConfig{
        SignerType: signer.PrivateKey,
        ChainID:    137,
        PrivateKeyConfig: &signer.PrivateKeyClient{
            PrivateKey: pk,
            Address:    common.HexToAddress("0xYOUR_EOA_ADDRESS"),
        },
    })
    if err != nil {
        log.Fatal(err)
    }

    // --- Optional builder config (needed for builder/relayer flows) ---
    builderCfg := &headers.BuilderConfig{
        APIKey:     "YOUR_BUILDER_API_KEY",
        Secret:     "YOUR_BUILDER_SECRET",
        Passphrase: "YOUR_BUILDER_PASSPHRASE",
    }

    // --- CLOB client ---
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

    // --- Read the orderbook ---
    ob, err := c.GetOrderBook("YOUR_TOKEN_ID")
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("asks=%d  bids=%d", len(ob.Asks), len(ob.Bids))
}
```

---

#### Step 2: Derive API Keys and Place a Limit Order

`DeriveApiKey` reuses existing L2 credentials instead of creating new ones (creation fails when a key already exists).

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
    // Assumes clobClient was initialized as shown in Step 1.
    var clobClient *clob.ClobClient

    // --- Derive L2 API credentials ---
    apiCreds, err := clobClient.DeriveApiKey(nil, clob_types.ClobOption{})
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("apiKey=%s", apiCreds.ApiKey)

    // --- Market info needed for order options ---
    tickSize := types.TickSize("0.01") // fetch from GetMarket in production
    negRisk  := false

    // --- Place a limit order (GTC, BUY 8 shares @ 0.15) ---
    resp, err := clobClient.CreateAndPostOrder(
        clob_types.OrderArgs{
            TokenID:    "YOUR_TOKEN_ID",
            Price:      decimal.RequireFromString("0.15"),
            Size:       decimal.RequireFromString("8"),
            Side:       types.SideBuy,
            FeeRateBps: 0,
            Nonce:      0,
            Expiration: 0,
        },
        clob_types.PartialCreateOrderOptions{
            OrderType: types.OrderTypeGTC,
            TickSize:  &tickSize,
            NegRisk:   &negRisk,
        },
    )
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("orderID=%s  status=%s", resp.OrderID, resp.Status)
}
```

> Before trading with real funds, verify market status, minimum tick size, fee rates, wallet balances, and token allowances.

---

#### Step 3: Place a Market Order

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
    // Assumes clobClient was initialized as shown in Step 1.
    var clobClient *clob.ClobClient

    tickSize := types.TickSize("0.01")
    negRisk  := false

    // --- Place a market order (buy $10 worth at ~0.60) ---
    resp, err := clobClient.CreateAndPostMarketOrder(
        clob_types.MarketOrderArgs{
            TokenID:    "YOUR_TOKEN_ID",
            Side:       types.SideBuy,
            Amount:     decimal.RequireFromString("10"),  // USDC amount
            Price:      decimal.RequireFromString("0.60"), // worst acceptable price
            FeeRateBps: 0,
            Nonce:      0,
            OrderType:  types.OrderTypeFOK,
        },
        clob_types.PartialCreateOrderOptions{
            TickSize: &tickSize,
            NegRisk:  &negRisk,
        },
    )
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("orderID=%s  status=%s", resp.OrderID, resp.Status)
}
```

---

#### Step 4: Subscribe to Real-Time Order Book via WebSocket

```go
package main

import (
    "log"
    "time"

    "github.com/ybina/polymarket-go/client/ws"
    "github.com/ybina/polymarket-go/client/types"
)

func main() {
    // Assumes clobClient was initialized as shown in Step 1.

    wsClient := ws.NewWebSocketClient(clobClient, &ws.WebSocketClientOptions{
        AutoReconnect:  true,
        ReconnectDelay: 5 * time.Second,
        ProxyUrl:       "", // optional
    })

    wsClient.On(&ws.WebSocketCallbacks{
        OnBook: func(msg *types.BookMessage) {
            log.Printf("book update: %d asks, %d bids", len(msg.Asks), len(msg.Bids))
        },
        OnError: func(err error) {
            log.Println("ws error:", err)
        },
    })

    if err := wsClient.Connect(); err != nil {
        log.Fatal(err)
    }
    if err := wsClient.Subscribe([]string{"YOUR_TOKEN_ID"}); err != nil {
        log.Fatal(err)
    }

    select {} // block forever
}
```

---

### Flow B — Turnkey Signer

#### Step 5: Initialize Turnkey and Create a Sub-Account

```go
package main

import (
    "log"

    "github.com/ybina/polymarket-go/turnkey"
)

func main() {
    tk, err := turnkey.NewTurnKeyService(turnkey.Config{
        PubKey:       "YOUR_TURNKEY_API_PUBLIC_KEY",
        PrivateKey:   "YOUR_TURNKEY_API_PRIVATE_KEY",
        Organization: "YOUR_TURNKEY_ORGANIZATION_ID",
        WalletName:   "YOUR_MASTER_WALLET_NAME",
    })
    if err != nil {
        log.Fatal(err)
    }

    // Derive the N-th address at BIP-32 path m/44'/60'/0'/0/N
    addr, err := tk.CreateAccount(1)
    if err != nil {
        log.Fatal(err)
    }
    log.Println("derived address:", addr)
}
```

To use Turnkey as the signer in any SDK client:

```go
sk, err := signer.NewSigner(signer.SignerConfig{
    SignerType: signer.Turnkey,
    ChainID:    137,
    TurnkeyConfig: &turnkey.Config{
        PubKey:       "YOUR_TURNKEY_API_PUBLIC_KEY",
        PrivateKey:   "YOUR_TURNKEY_API_PRIVATE_KEY",
        Organization: "YOUR_TURNKEY_ORGANIZATION_ID",
        WalletName:   "YOUR_MASTER_WALLET_NAME",
    },
})
if err != nil {
    log.Fatal(err)
}
```

---

#### Step 6: Deploy a Safe Wallet and Approve for Polymarket

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
        PubKey:       "YOUR_TURNKEY_API_PUBLIC_KEY",
        PrivateKey:   "YOUR_TURNKEY_API_PRIVATE_KEY",
        Organization: "YOUR_TURNKEY_ORGANIZATION_ID",
        WalletName:   "YOUR_MASTER_WALLET_NAME",
    }

    sk, err := signer.NewSigner(signer.SignerConfig{
        SignerType:    signer.Turnkey,
        TurnkeyConfig: &tkCfg,
        ChainID:       137,
    })
    if err != nil {
        log.Fatal(err)
    }

    builderCfg := headers.BuilderConfig{
        APIKey:     "YOUR_BUILDER_API_KEY",
        Secret:     "YOUR_BUILDER_SECRET",
        Passphrase: "YOUR_BUILDER_PASSPHRASE",
    }

    proxyURL := "" // e.g. "http://127.0.0.1:7890"
    rc, err := relayer.NewRelayClient(
        "https://relayer-v2.polymarket.com",
        types.ChainPolygon,
        sk,
        &builderCfg,
        &proxyURL,
    )
    if err != nil {
        log.Fatal(err)
    }

    turnkeyAccount := common.HexToAddress("0xYOUR_TURNKEY_ACCOUNT_ADDRESS")

    // --- Deploy Safe wallet ---
    safeAddr, deployResp, err := rc.DeployWithTurnkey(turnkeyAccount)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("[DEPLOY] safe=%s  txID=%s  txHash=%s\n",
        safeAddr.Hex(), deployResp.TransactionID, deployResp.TransactionHash)

    // --- Wait for on-chain confirmation (up to 90 s) ---
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
    fmt.Printf("[DEPLOY] confirmed: %s\n", safeAddr.Hex())

    // --- Approve the Safe wallet for Polymarket trading ---
    approveResp, err := rc.ApproveForPolymarketWithTurnkey(turnkeyAccount)
    if err != nil {
        log.Fatal(err)
    }
    if approveResp == nil {
        fmt.Println("[APPROVE] already approved; nothing to do")
        return
    }
    fmt.Printf("[APPROVE] txID=%s  txHash=%s\n",
        approveResp.TransactionID, approveResp.TransactionHash)
}
```

---

### Bridge: Create a Deposit Address

```go
package main

import (
    "log"
    "time"

    "github.com/bytedance/sonic"
    "github.com/ethereum/go-ethereum/common"

    "github.com/ybina/polymarket-go/client/bridge"
)

func main() {
    client, err := bridge.NewBridgeClient(&bridge.ClientConfig{
        Timeout:  20 * time.Second,
        ProxyUrl: "",
    })
    if err != nil {
        log.Fatal(err)
    }

    safeAddr := common.HexToAddress("0xYOUR_SAFE_ADDRESS")
    res, err := client.CreateDepositAddress(safeAddr)
    if err != nil {
        log.Fatal(err)
    }

    resStr, _ := sonic.MarshalString(res)
    log.Println(resStr)
}
```

---

## Testing

Test files live alongside each package (e.g. `client/clob/clob_client_test.go`, `client/gamma/client_test.go`, `turnkey/turnkeyService_test.go`).
All credential fields use `YOUR_*` placeholder strings—**never** hard-code real keys.

```bash
go test ./...
```

---

## License

MIT — see [`LICENSE`](LICENSE)

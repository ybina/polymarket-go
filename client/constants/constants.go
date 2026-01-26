package constants

import "github.com/ethereum/go-ethereum/common"

var (
	PolyExchangeDomainName = "Polymarket CTF Exchange"
	SAFE_FACTORY_NAME      = "Polymarket Contract Proxy Factory"
	SAFE_INIT_CODE_HASH    = "0x2bce2127ff07fb632d16c8347c4ebf501f4841168bed00d9e6ef715ddb6fcecf"
	ZERO_ADDRESS           = common.HexToAddress("0x0000000000000000000000000000000000000000")
	CTF_CONTRACT           = common.HexToAddress("0x4d97dcd97ec945f40cf65f87097ace5ea0476045")
	CTF_EXCHANGE           = common.HexToAddress("0x4bFb41d5B3570DeFd03C39a9A4D8dE6Bd8B8982E")
	NEGRISK_CTF            = common.HexToAddress("0xC5d563A36AE78145C45a50134d48A1215220f80a")
	NEGRISK_ADAPTER        = common.HexToAddress("0xd91E80cF2E7be2e162c6513ceD06f1dD0dA35296")
	USDCe                  = common.HexToAddress("0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174")
)

type AccessLevel uint8

var (
	L0 AccessLevel = 0
	L1 AccessLevel = 1
	L2 AccessLevel = 2
)

var (
	L1_AUTH_UNAVAILABLE = "a private key is needed to interact with this endpoint!"

	L2_AUTH_UNAVAILABLE = "API Credentials are needed to interact with this endpoint!"

	BUILDER_AUTH_UNAVAILABLE = "builder API Credentials needed to interact with this endpoint!"
)

type SigType int

const (
	// EOA ECDSA EIP712 signatures signed by EOAs
	EOA SigType = iota
	// POLY_PROXY EIP712 signatures signed by EOAs that own Polymarket Proxy wallets
	POLY_PROXY
	// POLY_GNOSIS_SAFE EIP712 signatures signed by EOAs that own Polymarket Gnosis safes
	POLY_GNOSIS_SAFE
)

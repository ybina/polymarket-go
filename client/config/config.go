package config

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ybina/polymarket-go/client/types"
)

type ContractConfig struct {
	SafeFactory          common.Address
	SafeMultisend        common.Address
	Exchange             common.Address
	NegExchange          common.Address
	Collateral           common.Address
	NegCollateral        common.Address
	ConditionalTokens    common.Address
	NegConditionalTokens common.Address
}

var contractConfigs = map[types.Chain]ContractConfig{
	137: {
		SafeFactory:          common.HexToAddress("0xaacFeEa03eb1561C4e67d661e40682Bd20E3541b"),
		SafeMultisend:        common.HexToAddress("0xA238CBeb142c10Ef7Ad8442C6D1f9E89e07e7761"),
		Exchange:             common.HexToAddress("0x4bFb41d5B3570DeFd03C39a9A4D8dE6Bd8B8982E"),
		NegExchange:          common.HexToAddress("0xC5d563A36AE78145C45a50134d48A1215220f80a"),
		Collateral:           common.HexToAddress("0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174"),
		NegCollateral:        common.HexToAddress("0x2791bca1f2de4661ed88a30c99a7a9449aa84174"),
		ConditionalTokens:    common.HexToAddress("0x4D97DCd97eC945f40cF65F87097ACe5EA0476045"),
		NegConditionalTokens: common.HexToAddress("0x4D97DCd97eC945f40cF65F87097ACe5EA0476045"),
	},
	80002: {
		SafeFactory:          common.HexToAddress("0xaacFeEa03eb1561C4e67d661e40682Bd20E3541b"),
		SafeMultisend:        common.HexToAddress("0xA238CBeb142c10Ef7Ad8442C6D1f9E89e07e7761"),
		Exchange:             common.HexToAddress("0xdFE02Eb6733538f8Ea35D585af8DE5958AD99E40"),
		NegExchange:          common.HexToAddress("0xd91E80cF2E7be2e162c6513ceD06f1dD0dA35296"),
		Collateral:           common.HexToAddress("0x9c4e1703476e875070ee25b56a58b008cfb8fa78"),
		NegCollateral:        common.HexToAddress("0x9c4e1703476e875070ee25b56a58b008cfb8fa78"),
		ConditionalTokens:    common.HexToAddress("0x69308FB512518e39F9b16112fA8d994F4e2Bf8bB"),
		NegConditionalTokens: common.HexToAddress("0x69308FB512518e39F9b16112fA8d994F4e2Bf8bB"),
	},
}

func GetContractConfig(chainID types.Chain) (ContractConfig, error) {
	cfg, ok := contractConfigs[chainID]
	if !ok {
		return ContractConfig{}, fmt.Errorf("invalid chainID: %d", chainID)
	}
	return cfg, nil
}

func GetWsPingInterval() time.Duration {
	return 10 * time.Second
}

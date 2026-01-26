package relayer

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/shopspring/decimal"
	"github.com/ybina/polymarket-go/tools/headers"

	"github.com/ybina/polymarket-go/client/config"
	"github.com/ybina/polymarket-go/client/constants"
	"github.com/ybina/polymarket-go/client/relayer/builder"
	"github.com/ybina/polymarket-go/client/relayer/model"
	"github.com/ybina/polymarket-go/client/signer"
	"github.com/ybina/polymarket-go/client/types"
)

type RelayClient struct {
	RelayerURL     string
	ChainID        types.Chain
	Signer         *signer.Signer
	BuilderConfig  *headers.BuilderConfig
	HttpClient     *http.Client
	ContractConfig config.ContractConfig
}

type RelayerTransaction struct {
	TransactionID   string `json:"transactionID"`
	TransactionHash string `json:"transactionHash"`
	State           string `json:"state"`
}

func NewRelayClient(
	relayerURL string,
	chainID types.Chain,
	signer *signer.Signer,
	builderConfig *headers.BuilderConfig,
	proxyUrl *string,

) (*RelayClient, error) {
	if strings.HasSuffix(relayerURL, "/") {
		relayerURL = strings.TrimRight(relayerURL, "/")
	}
	cfg, err := config.GetContractConfig(chainID)
	if err != nil {
		return nil, err
	}
	var transport *http.Transport
	if proxyUrl != nil && *proxyUrl != "" {
		proxyParsed, err := url.Parse(*proxyUrl)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy url: %w", err)
		}

		transport = &http.Transport{
			Proxy: http.ProxyURL(proxyParsed),
		}
	} else {
		transport = &http.Transport{}
	}

	return &RelayClient{
		RelayerURL:    relayerURL,
		ChainID:       chainID,
		Signer:        signer,
		BuilderConfig: builderConfig,
		HttpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
		ContractConfig: cfg,
	}, nil
}

func (c *RelayClient) GetNonce(address common.Address, signerType string) (uint64, error) {
	reqUrl := fmt.Sprintf("%s%s?address=%s&type=%s", c.RelayerURL, GET_NONCE, address.Hex(), signerType)

	resp, err := c.HttpClient.Get(reqUrl)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	if resp.StatusCode != http.StatusOK {

		return 0, fmt.Errorf("get nonce http %d: %s", resp.StatusCode, string(body))
	}

	var out struct {
		Nonce string `json:"nonce"`
	}
	err = sonic.Unmarshal(body, &out)
	if err != nil {
		return 0, err
	}
	nonce, err := strconv.ParseUint(out.Nonce, 10, 64)
	if err != nil {
		return 0, err
	}
	return nonce, nil
}

func (c *RelayClient) GetSafeNonceOnChain(safe common.Address) (uint64, error) {
	rpcURL := "https://polygon-rpc.com/"
	if strings.TrimSpace(rpcURL) == "" {
		return 0, fmt.Errorf("rpc url is empty")
	}

	cli, err := ethclient.Dial(rpcURL)
	if err != nil {
		return 0, fmt.Errorf("dial rpc failed: %w", err)
	}
	defer cli.Close()

	parsed, err := abi.JSON(strings.NewReader(safeNonceABI))
	if err != nil {
		return 0, fmt.Errorf("parse safe nonce abi failed: %w", err)
	}

	data, err := parsed.Pack("nonce")
	if err != nil {
		return 0, fmt.Errorf("pack nonce failed: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := cli.CallContract(ctx, ethereum.CallMsg{
		To:   &safe,
		Data: data,
	}, nil)
	if err != nil {
		return 0, fmt.Errorf("call nonce failed: %w", err)
	}
	if len(res) <= 0 {
		return 0, nil
	}
	out, err := parsed.Unpack("nonce", res)
	if err != nil {
		return 0, fmt.Errorf("unpack nonce failed: %w", err)
	}
	if len(out) != 1 {
		return 0, fmt.Errorf("unexpected nonce outputs len=%d", len(out))
	}

	nonceBig, ok := out[0].(*big.Int)
	if !ok {
		return 0, fmt.Errorf("nonce output type unexpected: %T", out[0])
	}
	if nonceBig.Sign() < 0 {
		return 0, fmt.Errorf("nonce is negative: %s", nonceBig.String())
	}
	if !nonceBig.IsUint64() {
		return 0, fmt.Errorf("nonce overflows uint64: %s", nonceBig.String())
	}

	return nonceBig.Uint64(), nil
}

func (c *RelayClient) GetTransaction(txID string) ([]RelayerTransaction, error) {
	reqUrl := fmt.Sprintf(
		"%s%s?id=%s",
		c.RelayerURL,
		GET_TRANSACTION,
		txID,
	)
	var resp []RelayerTransaction
	if err := c.getJSON(reqUrl, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *RelayClient) GetTransactions() ([]map[string]interface{}, error) {
	reqUrl := c.RelayerURL + GET_TRANSACTIONS
	return c.getJSONList(reqUrl)
}

func (c *RelayClient) getJSON(url string, out interface{}) error {
	resp, err := c.HttpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("http %d: %s", resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *RelayClient) getJSONList(url string) ([]map[string]interface{}, error) {
	resp, err := c.HttpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("http error")
	}

	var out []map[string]interface{}
	return out, json.NewDecoder(resp.Body).Decode(&out)
}

func (c *RelayClient) IsDeployed(safeAddr common.Address) (bool, error) {
	reqUrl := fmt.Sprintf("%s%s?address=%s", c.RelayerURL, GET_DEPLOYED, safeAddr.Hex())

	resp, err := c.HttpClient.Get(reqUrl)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("get deployed http %d: %s", resp.StatusCode, string(body))
	}

	var out struct {
		Deployed bool `json:"deployed"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return false, err
	}
	return out.Deployed, nil
}

func (c *RelayClient) ExecuteWithTurnkey(
	txs []model.SafeTransaction,
	metadata string,
	turnkeyAccount common.Address,
) (*ClientRelayerTransactionResponse, error) {

	if c.Signer == nil {
		return nil, errors.New("signer is required")
	}
	if c.BuilderConfig == nil {
		return nil, errors.New("builder config is required")
	}

	safeAddr := builder.Derive(turnkeyAccount, c.ContractConfig.SafeFactory)
	deployed, err := c.IsDeployed(safeAddr)
	if err != nil {
		return nil, err
	}
	if !deployed {
		return nil, fmt.Errorf("safe %s is not deployed", safeAddr)
	}
	nonce, err := c.GetSafeNonceOnChain(safeAddr)
	if err != nil {
		return nil, err
	}
	args := model.SafeTransactionArgs{
		FromAddress:  turnkeyAccount,
		Nonce:        nonce,
		ChainID:      c.ChainID,
		Transactions: txs,
	}

	req, err := builder.BuildSafeTransactionRequest(
		c.Signer,
		args,
		c.ContractConfig,
		metadata,
		turnkeyAccount,
	)

	if err != nil {
		return nil, err
	}
	return c.submit(req)
}

func (c *RelayClient) ExecuteWithPrivateKey(txs []model.SafeTransaction, metadata string) (*ClientRelayerTransactionResponse, error) {
	if c.Signer == nil {
		return nil, errors.New("signer is required")
	}
	if c.Signer.SignerType() != signer.PrivateKey {
		return nil, errors.New("signer is not a private key")
	}
	if c.BuilderConfig == nil {
		return nil, errors.New("builder config is required")
	}
	pub, err := c.Signer.GetPubkeyOfPrivateKey()
	if err != nil {
		return nil, err
	}
	safeAddr := builder.Derive(pub, c.ContractConfig.SafeFactory)
	deployed, err := c.IsDeployed(safeAddr)
	if err != nil {
		return nil, err
	}
	if !deployed {
		return nil, fmt.Errorf("safe %s is not deployed", safeAddr)
	}
	nonce, err := c.GetSafeNonceOnChain(safeAddr)
	if err != nil {
		return nil, err
	}
	args := model.SafeTransactionArgs{
		FromAddress:  pub,
		Nonce:        nonce,
		ChainID:      c.ChainID,
		Transactions: txs,
	}

	req, err := builder.BuildSafeTransactionRequest(
		c.Signer,
		args,
		c.ContractConfig,
		metadata,
		pub,
	)

	if err != nil {
		return nil, err
	}
	return c.submit(req)
}

func (c *RelayClient) DeployWithPrivateKey() (*ClientRelayerTransactionResponse, error) {
	if c.Signer == nil {
		return nil, errors.New("signer is required")
	}
	if c.Signer.SignerType() != signer.PrivateKey {
		return nil, errors.New("signer is not a private key")
	}

	if c.BuilderConfig == nil {
		return nil, errors.New("builder config is required")
	}
	pub, err := c.Signer.GetPubkeyOfPrivateKey()
	if err != nil {
		return nil, err
	}
	args := model.SafeCreateTransactionArgs{
		FromAddress:     pub,
		ChainID:         c.ChainID,
		PaymentToken:    constants.ZERO_ADDRESS,
		Payment:         "0",
		PaymentReceiver: constants.ZERO_ADDRESS,
	}
	req, err := builder.BuildSafeCreateTransactionRequest(
		c.Signer,
		args,
		c.ContractConfig,
		pub,
	)
	if err != nil {
		return nil, err
	}

	return c.submit(req)
}

// DeployWithTurnkey Deploy new Safe wallet
func (c *RelayClient) DeployWithTurnkey(turnkeyAccount common.Address) (common.Address, *ClientRelayerTransactionResponse, error) {
	if c.Signer == nil {
		return constants.ZERO_ADDRESS, nil, errors.New("signer is required")
	}

	if c.Signer.SignerType() != signer.Turnkey {
		return constants.ZERO_ADDRESS, nil, errors.New("signer is not a turnkey")
	}

	if c.BuilderConfig == nil {
		return constants.ZERO_ADDRESS, nil, errors.New("builder config is required")
	}

	safeAddr := builder.Derive(turnkeyAccount, c.ContractConfig.SafeFactory)
	log.Printf("builder.Derive SafeAddr: %s\n", safeAddr)

	args := model.SafeCreateTransactionArgs{
		FromAddress:     turnkeyAccount,
		ChainID:         c.ChainID,
		PaymentToken:    constants.ZERO_ADDRESS,
		Payment:         "0",
		PaymentReceiver: constants.ZERO_ADDRESS,
	}

	req, err := builder.BuildSafeCreateTransactionRequest(
		c.Signer,
		args,
		c.ContractConfig,
		turnkeyAccount,
	)
	if err != nil {
		return constants.ZERO_ADDRESS, nil, err
	}
	relayResp, err := c.submit(req)
	if err != nil {
		return constants.ZERO_ADDRESS, nil, err
	}
	return safeAddr, relayResp, nil
}

func (c *RelayClient) submit(req *model.TransactionRequest) (*ClientRelayerTransactionResponse, error) {
	raw, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	body := string(raw)

	builderHeaders, err := c.BuilderConfig.GenerateBuilderHeaders(
		"POST",
		SUBMIT_TRANSACTION,
		&body,
		"",
	)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest(
		"POST",
		c.RelayerURL+SUBMIT_TRANSACTION,
		bytes.NewReader(raw),
	)
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	httpReq.Header.Set("POLY_BUILDER_API_KEY", builderHeaders.POLYBuilderAPIKey)
	httpReq.Header.Set("POLY_BUILDER_TIMESTAMP", builderHeaders.POLYBuilderTimestamp)
	httpReq.Header.Set("POLY_BUILDER_PASSPHRASE", builderHeaders.POLYBuilderPassphrase)
	httpReq.Header.Set("POLY_BUILDER_SIGNATURE", builderHeaders.POLYBuilderSignature)

	resp, err := c.HttpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("relayer error %d: %s", resp.StatusCode, string(b))
	}
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if len(respBytes) > 0 {
		var pretty bytes.Buffer
		if err := json.Indent(&pretty, respBytes, "", "  "); err == nil {
			log.Printf("submit response (pretty):\n%s\n", pretty.String())
		} else {
			log.Printf("submit response (raw): %s\n", string(respBytes))
		}
	} else {
		log.Printf("submit response: <empty body>")
	}

	if len(respBytes) > 0 {
		var out struct {
			TransactionID   string `json:"transactionID"`
			TransactionHash string `json:"transactionHash"`
		}
		if err := json.Unmarshal(respBytes, &out); err != nil {
			return nil, err
		}

		return &ClientRelayerTransactionResponse{
			TransactionID:   out.TransactionID,
			TransactionHash: out.TransactionHash,
		}, nil
	}
	return &ClientRelayerTransactionResponse{}, nil
}

func (c *RelayClient) ApproveForPolymarketWithTurnkey(turnkeyAccount common.Address) (*ClientRelayerTransactionResponse, error) {
	if c.Signer == nil {
		return nil, errors.New("signer is required")
	}
	if c.Signer.SignerType() != signer.Turnkey {
		return nil, errors.New("signer is not a turnkey")
	}
	if turnkeyAccount == constants.ZERO_ADDRESS {
		return nil, errors.New("turnkey account is required")
	}
	safe := builder.Derive(turnkeyAccount, c.ContractConfig.SafeFactory)
	approved, usdcApprovals, tokenApprovals, err := c.CheckAllApprovals(safe)
	fmt.Printf("approved:%v, usdcApprove:%v, tokenApprove:%v \n", approved, usdcApprovals, tokenApprovals)
	if err != nil {
		return nil, err
	}
	if approved {
		return nil, nil
	}
	var txs []model.SafeTransaction
	if !usdcApprovals {
		usdc := constants.USDCe

		usdcSpenders := []common.Address{
			constants.CTF_CONTRACT,
			constants.NEGRISK_ADAPTER,
			constants.CTF_EXCHANGE,
			constants.NEGRISK_CTF,
		}

		for _, spender := range usdcSpenders {
			approveTxs, err := c.createUsdcApproveTxn(usdc, spender)
			if err != nil {
				return nil, err
			}
			txs = append(txs, approveTxs...)
		}
	}

	if !tokenApprovals {
		erc1155Operators := []common.Address{
			constants.CTF_EXCHANGE,
			constants.NEGRISK_CTF,
			constants.NEGRISK_ADAPTER,
		}

		for _, op := range erc1155Operators {
			erc1155Txs, err := c.createErc1155ApproveAllTxn(constants.CTF_CONTRACT, op, true)
			if err != nil {
				return nil, err
			}
			txs = append(txs, erc1155Txs...)
		}
	}
	var metadata string
	if !usdcApprovals && tokenApprovals {
		metadata = "Set all token approvals for trading"
	} else {
		if !usdcApprovals {
			metadata = "approve USDC to polymarket contracts"
		} else {
			// TODO: only approve erc1155 token will fail
			metadata = ""
		}
	}

	return c.ExecuteWithTurnkey(
		txs,
		metadata,
		turnkeyAccount,
	)
}

func (c *RelayClient) ApproveRequestContractForPolymarketWithTurnkey(
	turnkeyAccount common.Address,
	contractName string,
) (*ClientRelayerTransactionResponse, error) {

	if c.Signer == nil {
		return nil, errors.New("signer is required")
	}
	if c.Signer.SignerType() != signer.Turnkey {
		return nil, errors.New("signer is not a turnkey")
	}
	if turnkeyAccount == constants.ZERO_ADDRESS {
		return nil, errors.New("turnkey account is required")
	}

	var (
		usdcSpenders     []common.Address
		erc1155Operators []common.Address
	)

	switch strings.ToLower(strings.TrimSpace(contractName)) {

	case "ctf_contract":
		usdcSpenders = []common.Address{
			constants.CTF_CONTRACT,
		}
		erc1155Operators = []common.Address{
			constants.CTF_CONTRACT,
		}

	case "ctf_exchange":

		usdcSpenders = []common.Address{
			constants.CTF_EXCHANGE,
		}
		erc1155Operators = []common.Address{
			constants.CTF_EXCHANGE,
		}

	case "negrisk_ctf":

		usdcSpenders = []common.Address{
			constants.NEGRISK_CTF,
		}
		erc1155Operators = []common.Address{
			constants.NEGRISK_CTF,
		}

	case "negrisk_adapter":
		usdcSpenders = []common.Address{
			constants.NEGRISK_ADAPTER,
		}
		erc1155Operators = []common.Address{
			constants.NEGRISK_ADAPTER,
		}

	case "all":

		usdcSpenders = []common.Address{
			constants.CTF_CONTRACT,
			constants.CTF_EXCHANGE,
			constants.NEGRISK_ADAPTER,
			constants.NEGRISK_CTF,
		}
		erc1155Operators = []common.Address{
			constants.CTF_CONTRACT,
			constants.CTF_EXCHANGE,
			constants.NEGRISK_CTF,
			constants.NEGRISK_ADAPTER,
		}

	default:
		return nil, fmt.Errorf("unknown contractName: %s", contractName)
	}

	var txs []model.SafeTransaction

	usdc := constants.USDCe
	for _, spender := range usdcSpenders {
		approveTxs, err := c.createUsdcApproveTxn(usdc, spender)
		if err != nil {
			return nil, err
		}
		txs = append(txs, approveTxs...)
	}

	for _, op := range erc1155Operators {
		erc1155Txs, err := c.createErc1155ApproveAllTxn(constants.CTF_CONTRACT, op, true)
		if err != nil {
			return nil, err
		}
		txs = append(txs, erc1155Txs...)
	}

	if len(txs) == 0 {
		return nil, nil
	}

	metadata := fmt.Sprintf("approve USDC to polymarket contracts")

	return c.ExecuteWithTurnkey(
		txs,
		metadata,
		turnkeyAccount,
	)
}

func (c *RelayClient) functionSelector(signature string) []byte {
	return crypto.Keccak256([]byte(signature))[:4]
}

func (c *RelayClient) createUsdcApproveTxn(
	usdceAddr common.Address,
	contractAddr common.Address,
) ([]model.SafeTransaction, error) {

	maxUint := new(big.Int).Sub(
		new(big.Int).Lsh(big.NewInt(1), 256),
		big.NewInt(1),
	)

	data, err := c.encodeApprove(contractAddr, maxUint)
	if err != nil {
		return nil, err
	}

	txn := model.SafeTransaction{
		To:        usdceAddr,
		Operation: model.Call,
		Data:      data,
		Value:     "0",
	}

	return []model.SafeTransaction{txn}, nil
}

func (c *RelayClient) encodeApprove(spender common.Address, amount *big.Int) (string, error) {
	selector := c.functionSelector("approve(address,uint256)")

	addressType, _ := abi.NewType("address", "", nil)
	uintType, _ := abi.NewType("uint256", "", nil)

	args := abi.Arguments{
		{Type: addressType},
		{Type: uintType},
	}

	encodedArgs, err := args.Pack(spender, amount)
	if err != nil {
		return "", err
	}

	data := append(selector, encodedArgs...)
	return "0x" + hex.EncodeToString(data), nil
}

func (c *RelayClient) createErc1155ApproveAllTxn(
	ctfContract common.Address,
	operator common.Address,
	approved bool,
) ([]model.SafeTransaction, error) {

	data, err := c.encodeSetApprovalForAll(operator, approved)
	if err != nil {
		return nil, err
	}

	txn := model.SafeTransaction{
		To:        ctfContract,
		Operation: model.Call,
		Data:      data,
		Value:     "0",
	}

	return []model.SafeTransaction{txn}, nil
}

func (c *RelayClient) encodeSetApprovalForAll(operator common.Address, approved bool) (string, error) {
	selector := c.functionSelector("setApprovalForAll(address,bool)")

	addressType, _ := abi.NewType("address", "", nil)
	boolType, _ := abi.NewType("bool", "", nil)

	args := abi.Arguments{
		{Type: addressType},
		{Type: boolType},
	}

	encodedArgs, err := args.Pack(operator, approved)
	if err != nil {
		return "", err
	}

	data := append(selector, encodedArgs...)

	return "0x" + hex.EncodeToString(data), nil
}

func (c *RelayClient) CheckUsdcApprovalForSpender(safeAddr common.Address, spender common.Address) (bool, error) {
	rpcURL := "https://polygon-rpc.com/"
	if strings.TrimSpace(rpcURL) == "" {
		return false, fmt.Errorf("rpc url is empty (ContractConfig.RpcUrl)")
	}

	cli, err := ethclient.Dial(rpcURL)
	if err != nil {
		return false, fmt.Errorf("dial rpc failed: %w", err)
	}
	defer cli.Close()

	parsed, err := abi.JSON(strings.NewReader(erc20AllowanceABI))
	if err != nil {
		return false, fmt.Errorf("parse erc20 abi failed: %w", err)
	}

	data, err := parsed.Pack("allowance", safeAddr, spender)
	if err != nil {
		return false, fmt.Errorf("pack allowance failed: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := cli.CallContract(ctx, ethereum.CallMsg{
		To:   &constants.USDCe,
		Data: data,
	}, nil)
	if err != nil {
		return false, fmt.Errorf("call allowance failed: %w", err)
	}

	out, err := parsed.Unpack("allowance", res)
	if err != nil {
		return false, fmt.Errorf("unpack allowance failed: %w", err)
	}
	if len(out) != 1 {
		return false, fmt.Errorf("unexpected allowance outputs len=%d", len(out))
	}

	allowance, ok := out[0].(*big.Int)
	if !ok {
		return false, fmt.Errorf("allowance output type unexpected: %T", out[0])
	}

	threshold := new(big.Int)
	threshold.SetString("1000000000000", 10) // 1e12
	return allowance.Cmp(threshold) >= 0, nil
}

func (c *RelayClient) CheckERC1155ApprovalForSpender(safeAddr common.Address, spender common.Address) (bool, error) {
	rpcURL := "https://polygon-rpc.com/"
	if strings.TrimSpace(rpcURL) == "" {
		return false, fmt.Errorf("rpc url is empty (ContractConfig.RpcUrl)")
	}

	cli, err := ethclient.Dial(rpcURL)
	if err != nil {
		return false, fmt.Errorf("dial rpc failed: %w", err)
	}
	defer cli.Close()

	parsed, err := abi.JSON(strings.NewReader(erc1155ApprovalABI))
	if err != nil {
		return false, fmt.Errorf("parse erc1155 abi failed: %w", err)
	}

	data, err := parsed.Pack("isApprovedForAll", safeAddr, spender)
	if err != nil {
		return false, fmt.Errorf("pack isApprovedForAll failed: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := cli.CallContract(ctx, ethereum.CallMsg{
		To:   &constants.CTF_CONTRACT,
		Data: data,
	}, nil)
	if err != nil {
		return false, fmt.Errorf("call isApprovedForAll failed: %w", err)
	}

	out, err := parsed.Unpack("isApprovedForAll", res)
	if err != nil {
		return false, fmt.Errorf("unpack isApprovedForAll failed: %w", err)
	}
	if len(out) != 1 {
		return false, fmt.Errorf("unexpected isApprovedForAll outputs len=%d", len(out))
	}

	approved, ok := out[0].(bool)
	if !ok {
		return false, fmt.Errorf("isApprovedForAll output type unexpected: %T", out[0])
	}
	return approved, nil
}

func (c *RelayClient) CheckAllApprovals(
	safeAddr common.Address,
) (allApproved, usdcApprovals, outcomeTokenApprovals bool, err error) {

	usdcSpenders := []common.Address{
		constants.CTF_CONTRACT,
		constants.NEGRISK_ADAPTER,
		constants.CTF_EXCHANGE,
		constants.NEGRISK_CTF,
	}

	usdcAll := true
	for _, spender := range usdcSpenders {
		ok, e := c.CheckUsdcApprovalForSpender(safeAddr, spender)
		if e != nil {
			return false, false, false, e
		}
		if !ok {
			usdcAll = false
		}
	}

	erc1155Operators := []common.Address{
		constants.CTF_EXCHANGE,
		constants.NEGRISK_CTF,
		constants.NEGRISK_ADAPTER,
	}

	outcomeAll := true
	for _, op := range erc1155Operators {
		ok, e := c.CheckERC1155ApprovalForSpender(safeAddr, op)
		if e != nil {
			return false, false, false, e
		}
		if !ok {
			outcomeAll = false
		}
	}

	usdcApprovals = usdcAll
	outcomeTokenApprovals = outcomeAll
	allApproved = usdcAll && outcomeAll
	return allApproved, usdcApprovals, outcomeTokenApprovals, nil
}

func (c *RelayClient) TransferUsdceFromSafeWithTurnkey(
	turnkeyAccount common.Address,
	targetAddr common.Address,
	amount decimal.Decimal,
) (string, error) {
	if c.Signer == nil {
		return "", errors.New("signer is required")
	}
	if c.Signer.SignerType() != signer.Turnkey {
		return "", errors.New("signer is not a turnkey")
	}
	if c.BuilderConfig == nil {
		return "", errors.New("builder config is required")
	}
	if turnkeyAccount == constants.ZERO_ADDRESS {
		return "", errors.New("turnkey account is required")
	}
	if targetAddr == constants.ZERO_ADDRESS {
		return "", errors.New("target address is required")
	}
	if amount.LessThanOrEqual(decimal.Zero) {
		return "", fmt.Errorf("amount must be > 0, got %s", amount.String())
	}

	scaled := amount.Shift(6).Truncate(0)
	amtBig := scaled.BigInt()

	if amtBig.Sign() <= 0 {
		return "", fmt.Errorf("amount too small after truncation (USDCe 6 decimals): %s", amount.String())
	}

	data, err := c.encodeTransfer(targetAddr, amtBig)
	if err != nil {
		return "", err
	}

	txs := []model.SafeTransaction{
		{
			To:        constants.USDCe,
			Operation: model.Call,
			Data:      data,
			Value:     "0",
		},
	}

	metadata := fmt.Sprintf("Transfer USDC.e to %s", targetAddr.Hex())

	resp, err := c.ExecuteWithTurnkey(txs, metadata, turnkeyAccount)
	if err != nil {
		return "", err
	}
	if resp != nil {
		log.Printf("Transfer submitted. txID=%s txHash=%s", resp.TransactionID, resp.TransactionHash)
		return resp.TransactionHash, nil
	}
	return "", errors.New("response is empty")
}

func (c *RelayClient) encodeTransfer(to common.Address, amount *big.Int) (string, error) {
	selector := c.functionSelector("transfer(address,uint256)")

	addressType, _ := abi.NewType("address", "", nil)
	uintType, _ := abi.NewType("uint256", "", nil)

	args := abi.Arguments{
		{Type: addressType},
		{Type: uintType},
	}

	encodedArgs, err := args.Pack(to, amount)
	if err != nil {
		return "", err
	}

	data := append(selector, encodedArgs...)
	return "0x" + hex.EncodeToString(data), nil
}

func (c *RelayClient) PollUntilState(
	transactionID string,
	states []model.RelayerTransactionState,
	failState model.RelayerTransactionState,
	maxPolls int,
	pollInterval time.Duration,
) (*RelayerTransaction, error) {

	stateSet := make(map[model.RelayerTransactionState]struct{})
	for _, s := range states {
		stateSet[s] = struct{}{}
	}

	for i := 0; i < maxPolls; i++ {

		txns, err := c.GetTransaction(transactionID)
		if err != nil {
			return nil, err
		}

		if len(txns) > 0 {
			txn := txns[0]
			state := model.RelayerTransactionState(txn.State)

			if _, ok := stateSet[state]; ok {
				return &txn, nil
			}

			if state == failState {
				return nil, nil
			}
		}

		time.Sleep(pollInterval)
	}

	return nil, nil
}

var (
	ctfABIOnce   sync.Once
	ctfABIParsed abi.ABI
	ctfABIErr    error
)

func getCTFABI() (abi.ABI, error) {
	ctfABIOnce.Do(func() {
		ctfABIParsed, ctfABIErr = abi.JSON(strings.NewReader(ctfABI))
	})
	return ctfABIParsed, ctfABIErr
}

func toBigIntSlice(xs []uint64) []*big.Int {
	out := make([]*big.Int, 0, len(xs))
	for _, v := range xs {
		out = append(out, new(big.Int).SetUint64(v))
	}
	return out
}

func (c *RelayClient) packCTF(method string, args ...interface{}) (string, error) {
	parsed, err := getCTFABI()
	if err != nil {
		return "", fmt.Errorf("parse ctf abi failed: %w", err)
	}
	data, err := parsed.Pack(method, args...)
	if err != nil {
		return "", fmt.Errorf("pack %s failed: %w", method, err)
	}
	return "0x" + hex.EncodeToString(data), nil
}

func (c *RelayClient) executeSafeTxs(txs []model.SafeTransaction, metadata string, turnkeyAccount common.Address) (*ClientRelayerTransactionResponse, error) {
	if c.Signer == nil {
		return nil, errors.New("signer is required")
	}
	switch c.Signer.SignerType() {
	case signer.Turnkey:
		if turnkeyAccount == constants.ZERO_ADDRESS {
			return nil, errors.New("turnkey account is required")
		}
		return c.ExecuteWithTurnkey(txs, metadata, turnkeyAccount)
	case signer.PrivateKey:
		return c.ExecuteWithPrivateKey(txs, metadata)
	default:
		return nil, fmt.Errorf("unsupported signer type: %v", c.Signer.SignerType())
	}
}

// Redeem positions:
// redeemPositions(collateralToken, parentCollectionId, conditionId, indexSets)
func (c *RelayClient) RedeemPosition(
	turnkeyAccount common.Address,
	collateralToken common.Address,
	parentCollectionId common.Hash,
	conditionId common.Hash,
	indexSets []uint64,
) (*ClientRelayerTransactionResponse, error) {

	if collateralToken == constants.ZERO_ADDRESS {
		return nil, errors.New("collateralToken is required")
	}
	if (conditionId == common.Hash{}) {
		return nil, errors.New("conditionId is required")
	}
	if len(indexSets) == 0 {
		return nil, errors.New("indexSets is required")
	}

	data, err := c.packCTF(
		"redeemPositions",
		collateralToken,
		parentCollectionId,
		conditionId,
		toBigIntSlice(indexSets),
	)
	if err != nil {
		return nil, err
	}

	txs := []model.SafeTransaction{
		{
			To:        constants.CTF_CONTRACT,
			Operation: model.Call,
			Data:      data,
			Value:     "0",
		},
	}

	return c.executeSafeTxs(txs, "Redeem positions", turnkeyAccount)
}

// Split positions:
// splitPosition(collateralToken, parentCollectionId, conditionId, partition, amount)
func (c *RelayClient) SplitPosition(
	turnkeyAccount common.Address,
	collateralToken common.Address,
	parentCollectionId common.Hash,
	conditionId common.Hash,
	partition []uint64,
	amount *big.Int,
) (*ClientRelayerTransactionResponse, error) {

	if collateralToken == constants.ZERO_ADDRESS {
		return nil, errors.New("collateralToken is required")
	}
	if (conditionId == common.Hash{}) {
		return nil, errors.New("conditionId is required")
	}
	if len(partition) == 0 {
		return nil, errors.New("partition is required")
	}
	if amount == nil || amount.Sign() <= 0 {
		return nil, errors.New("amount must be > 0")
	}

	data, err := c.packCTF(
		"splitPosition",
		collateralToken,
		parentCollectionId,
		conditionId,
		toBigIntSlice(partition),
		amount,
	)
	if err != nil {
		return nil, err
	}

	txs := []model.SafeTransaction{
		{
			To:        constants.CTF_CONTRACT,
			Operation: model.Call,
			Data:      data,
			Value:     "0",
		},
	}

	return c.executeSafeTxs(txs, "Split positions", turnkeyAccount)
}

// Merge positions:
// mergePositions(collateralToken, parentCollectionId, conditionId, partition, amount)
func (c *RelayClient) MergePosition(
	turnkeyAccount common.Address,
	collateralToken common.Address,
	parentCollectionId common.Hash,
	conditionId common.Hash,
	partition []uint64,
	amount *big.Int,
) (*ClientRelayerTransactionResponse, error) {

	if collateralToken == constants.ZERO_ADDRESS {
		return nil, errors.New("collateralToken is required")
	}
	if (conditionId == common.Hash{}) {
		return nil, errors.New("conditionId is required")
	}
	if len(partition) == 0 {
		return nil, errors.New("partition is required")
	}
	if amount == nil || amount.Sign() <= 0 {
		return nil, errors.New("amount must be > 0")
	}

	data, err := c.packCTF(
		"mergePositions",
		collateralToken,
		parentCollectionId,
		conditionId,
		toBigIntSlice(partition),
		amount,
	)
	if err != nil {
		return nil, err
	}

	txs := []model.SafeTransaction{
		{
			To:        constants.CTF_CONTRACT,
			Operation: model.Call,
			Data:      data,
			Value:     "0",
		},
	}

	return c.executeSafeTxs(txs, "Merge positions", turnkeyAccount)
}

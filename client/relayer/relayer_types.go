package relayer

type ClientRelayerTransactionResponse struct {
	TransactionID   string `json:"transactionID"`
	TransactionHash string `json:"transactionHash"`
	Hash            string `json:"hash"`
}

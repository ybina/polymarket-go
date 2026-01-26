package relayer

const erc20AllowanceABI = `[{
  "inputs":[{"name":"owner","type":"address"},{"name":"spender","type":"address"}],
  "name":"allowance",
  "outputs":[{"name":"","type":"uint256"}],
  "stateMutability":"view",
  "type":"function"
}]`

const erc1155ApprovalABI = `[{
  "inputs":[{"name":"account","type":"address"},{"name":"operator","type":"address"}],
  "name":"isApprovedForAll",
  "outputs":[{"name":"","type":"bool"}],
  "stateMutability":"view",
  "type":"function"
}]`

const safeNonceABI = `[{
  "inputs": [],
  "name": "nonce",
  "outputs": [{"name": "", "type": "uint256"}],
  "stateMutability": "view",
  "type": "function"
}]`

const ctfABI = `[
  {
    "name": "redeemPositions",
    "type": "function",
    "inputs": [
      {"name":"collateralToken","type":"address"},
      {"name":"parentCollectionId","type":"bytes32"},
      {"name":"conditionId","type":"bytes32"},
      {"name":"indexSets","type":"uint256[]"}
    ],
    "outputs": []
  },
  {
    "name": "splitPosition",
    "type": "function",
    "inputs": [
      {"name":"collateralToken","type":"address"},
      {"name":"parentCollectionId","type":"bytes32"},
      {"name":"conditionId","type":"bytes32"},
      {"name":"partition","type":"uint256[]"},
      {"name":"amount","type":"uint256"}
    ],
    "outputs": []
  },
  {
    "name": "mergePositions",
    "type": "function",
    "inputs": [
      {"name":"collateralToken","type":"address"},
      {"name":"parentCollectionId","type":"bytes32"},
      {"name":"conditionId","type":"bytes32"},
      {"name":"partition","type":"uint256[]"},
      {"name":"amount","type":"uint256"}
    ],
    "outputs": []
  }
]`

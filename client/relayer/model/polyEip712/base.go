package polyEip712

type EIP712Type interface {
	TypeName() string
	Encode(value any) ([32]byte, error)
}

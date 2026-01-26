package polyEip712

type ArrayType struct {
	Member EIP712Type
}

func (a ArrayType) TypeName() string {
	return a.Member.TypeName() + "[]"
}

func (a ArrayType) Encode(value any) ([32]byte, error) {
	items := value.([]any)
	buf := []byte{}

	for _, v := range items {
		enc, err := a.Member.Encode(v)
		if err != nil {
			return [32]byte{}, err
		}
		buf = append(buf, enc[:]...)
	}
	return keccak(buf), nil
}

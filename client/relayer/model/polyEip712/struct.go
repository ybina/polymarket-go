package polyEip712

import (
	"bytes"
)

type StructMember struct {
	Name string
	Type EIP712Type
}

type EIP712Struct struct {
	TypeName string
	Members  []StructMember
	Values   map[string]any
}

func NewStruct(typeName string, members []StructMember, values map[string]any) *EIP712Struct {
	return &EIP712Struct{
		TypeName: typeName,
		Members:  members,
		Values:   values,
	}
}

func (s *EIP712Struct) EncodeData() ([]byte, error) {
	buf := []byte{}
	for _, m := range s.Members {
		val := s.Values[m.Name]
		enc, err := m.Type.Encode(val)
		if err != nil {
			return nil, err
		}
		buf = append(buf, enc[:]...)
	}
	return buf, nil
}

func (s *EIP712Struct) EncodeType() string {
	fields := []string{}
	for _, m := range s.Members {
		fields = append(fields, m.Type.TypeName()+" "+m.Name)
	}
	return s.TypeName + "(" + join(fields, ",") + ")"
}

func (s *EIP712Struct) TypeHash() [32]byte {
	return keccak([]byte(s.EncodeType()))
}

func (s *EIP712Struct) HashStruct() ([32]byte, error) {
	data, err := s.EncodeData()
	if err != nil {
		return [32]byte{}, err
	}
	th := s.TypeHash()
	buf := append(th[:], data...)
	return keccak(buf), nil
}

func (s *EIP712Struct) SignableBytes(domain *EIP712Struct) ([]byte, error) {
	dh, err := domain.HashStruct()
	if err != nil {
		return nil, err
	}
	sh, err := s.HashStruct()
	if err != nil {
		return nil, err
	}
	return bytes.Join([][]byte{
		{0x19, 0x01},
		dh[:],
		sh[:],
	}, nil), nil
}

func join(ss []string, sep string) string {
	if len(ss) == 0 {
		return ""
	}
	out := ss[0]
	for i := 1; i < len(ss); i++ {
		out += sep + ss[i]
	}
	return out
}

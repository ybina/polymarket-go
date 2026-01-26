package polyEip712

func MakeDomain(
	name *string,
	version *string,
	chainId *int,
	verifyingContract *string,
	salt *string,
) *EIP712Struct {

	members := []StructMember{}
	values := map[string]any{}

	if name != nil {
		members = append(members, StructMember{"name", StringType{}})
		values["name"] = *name
	}
	if version != nil {
		members = append(members, StructMember{"version", StringType{}})
		values["version"] = *version
	}
	if chainId != nil {
		members = append(members, StructMember{"chainId", UintType{Bits: 256}})
		values["chainId"] = *chainId
	}
	if verifyingContract != nil {
		members = append(members, StructMember{"verifyingContract", AddressType{}})
		values["verifyingContract"] = *verifyingContract
	}
	if salt != nil {
		members = append(members, StructMember{"salt", BytesType{Length: 32}})
		values["salt"] = *salt
	}

	return NewStruct("EIP712Domain", members, values)
}

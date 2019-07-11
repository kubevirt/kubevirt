// +build gofuzz

package red

func Fuzz(data []byte) int {
	//return fuzzClientAuthMethod(data)
	//return fuzzClientLinkMessage(data)
	//return fuzzClientTicket(data)
	//return fuzzLinkHeader(data)
	//return fuzzMiniDataHeader(data)
	return fuzzServerLinkMessage(data)
	//return fuzzServerTicket(data)
}

func fuzzClientAuthMethod(data []byte) int {
	m := &ClientAuthMethod{}
	if err := (m).UnmarshalBinary(data); err != nil {
		return 0
	}

	if _, err := m.MarshalBinary(); err != nil {
		panic(err)
	}

	return 1
}

func fuzzClientLinkMessage(data []byte) int {
	m := &ClientLinkMessage{}
	if err := (m).UnmarshalBinary(data); err != nil {
		return 0
	}

	if _, err := m.MarshalBinary(); err != nil {
		panic(err)
	}

	return 1
}

func fuzzClientTicket(data []byte) int {
	m := &ClientTicket{}
	if err := (m).UnmarshalBinary(data); err != nil {
		return 0
	}

	if _, err := m.MarshalBinary(); err != nil {
		panic(err)
	}

	return 1
}

func fuzzLinkHeader(data []byte) int {
	m := &LinkHeader{}
	if err := (m).UnmarshalBinary(data); err != nil {
		return 0
	}

	if _, err := m.MarshalBinary(); err != nil {
		panic(err)
	}

	return 1
}

func fuzzMiniDataHeader(data []byte) int {
	m := &MiniDataHeader{}
	if err := (m).UnmarshalBinary(data); err != nil {
		return 0
	}

	if _, err := m.MarshalBinary(); err != nil {
		panic(err)
	}

	return 1
}

func fuzzServerLinkMessage(data []byte) int {
	m := &ServerLinkMessage{}
	if err := (m).UnmarshalBinary(data); err != nil {
		return 0
	}

	if _, err := m.MarshalBinary(); err != nil {
		panic(err)
	}

	return 1
}

func fuzzServerTicket(data []byte) int {
	m := &ServerTicket{}
	if err := (m).UnmarshalBinary(data); err != nil {
		return 0
	}

	if _, err := m.MarshalBinary(); err != nil {
		panic(err)
	}

	return 1
}

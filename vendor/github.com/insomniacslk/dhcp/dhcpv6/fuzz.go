// +build gofuzz

package dhcpv6

import (
	"bytes"
	"fmt"
)

// Fuzz is an entrypoint for go-fuzz (github.com/dvyukov/go-fuzz)
func Fuzz(data []byte) int {
	msg, err := FromBytes(data)
	if err != nil {
		return 0
	}

	serialized := msg.ToBytes()
	if !bytes.Equal(data, serialized) {
		rtMsg, err := FromBytes(serialized)
		fmt.Printf("Input:      %x\n", data)
		fmt.Printf("Round-trip: %x\n", serialized)
		fmt.Println("Message: ", msg.Summary())
		fmt.Printf("Go repr: %#v\n", msg)
		fmt.Println("round-trip reserialized: ", rtMsg.Summary())
		fmt.Printf("Go repr: %#v\n", rtMsg)
		if err != nil {
			fmt.Printf("failed to parse after deserialize-serialize: %v\n", err)
		}
		panic("round-trip different")
	}

	return 1
}

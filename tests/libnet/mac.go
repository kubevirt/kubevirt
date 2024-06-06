package libnet

import (
	cryptorand "crypto/rand"
	"net"
)

func GenerateRandomMac() (net.HardwareAddr, error) {
	prefix := net.HardwareAddr{0x02, 0x00, 0x00} // local unicast prefix
	const macByteSize = 3
	suffix := make(net.HardwareAddr, macByteSize)
	_, err := cryptorand.Read(suffix)
	if err != nil {
		return nil, err
	}
	return append(prefix, suffix...), nil
}

package tuntap

import (
	"fmt"
	"hash/crc32"
)

// makeRandomMac returns a random MAC address that is valid for a virtual machine.
func makeRandomMac(input string) string {
	// 00:16:3e is the prefix for qemu
	// get 3 random bytes:

	b1, b2, b3 := stringToBytes(input)
	return fmt.Sprintf("00:16:3e:%02x:%02x:%02x", b1, b2, b3) // qemu mac address prefix
}

func stringToBytes(input string) (byte, byte, byte) {
	// Compute CRC32 hash of the input
	hash := crc32.ChecksumIEEE([]byte(input))
	hashBytes := make([]byte, 4)

	// Convert hash to bytes
	hashBytes[0] = byte(hash >> 24)
	hashBytes[1] = byte(hash >> 16)
	hashBytes[2] = byte(hash >> 8)
	hashBytes[3] = byte(hash)

	// Return the first 3 bytes
	return hashBytes[0], hashBytes[1], hashBytes[2]
}

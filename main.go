package main

import (
	"context"
	"fmt"
	"hash/crc32"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	err := run(ctx, os.Stdout, os.Args, os.Environ())
	if err != nil {
		os.Exit(1)
	}
}

func run(ctx context.Context, stdout *os.File, args []string, env []string) error {

	return nil
}

func makeCommandLine(filename string) []string {
	var ext string
	if strings.HasSuffix(filename, ".qcow2") {
		ext = "qcow2"
	} else {
		ext = "raw"
	}
	return []string{
		"-drive", fmt.Sprintf("file=%s,format=%s", filename, ext),
		"-m", "512",
		"-netdev", getNativeNetworking(),
		"-device", "virtio-net,netdev=usernet,mac=" + generateMac(filename),
		"-nographic",
	}
}

// getNativeNetworking returns the correct -netdev string for the current OS
func getNativeNetworking() string {
	switch runtime.GOOS {
	case "darwin":
		return "vmnet-shared,id=net0"
	case "linux":
		return "tap,id=net0,ifname=tap0,br=br0,script=no"
	default:
		panic("unsupported OS")
	}
}

func generateMac(input ...string) string {
	input = append(input, os.Getenv("USER")) // add username to the input
	joined := strings.Join(input, "")
	if len(joined) == 0 {
		panic("no input to generate mac")
	}
	// use crc32 to generate a 32-bit hash
	hash := crc32.ChecksumIEEE([]byte(joined))
	// use qemu prefix 52:54
	return macFromInt("52:54", hash)
}

// generateTapName generates a tap device name from the input strings
// It isn't used atm.
func generateTapName(input ...string) string {
	input = append(input, os.Getenv("USER")) // add username to the input
	joined := strings.Join(input, "")
	if len(joined) == 0 {
		panic("no input to generate tap name")
	}
	// use crc32 to generate a 32-bit hash
	hash := crc32.ChecksumIEEE([]byte(joined))
	return fmt.Sprintf("tap%d", hash)
}

func macFromInt(prefix string, i uint32) string {
	// split into bytes:
	b1, b2, b3, b4 := byte(i>>24), byte(i>>16), byte(i>>8), byte(i)
	return fmt.Sprintf("%s:%02X:%02X:%02X", prefix, b1, b2, b3, b4)
}

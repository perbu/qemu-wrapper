package main

import (
	"context"
	"fmt"
	"hash/crc32"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
)

const (
	qemuBinary = "qemu-system-x86_64"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	err := run(ctx, os.Args, os.Environ())
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, env []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: %s <image>", args[0])
	}
	firmwarePath := args[1]
	cmd := exec.CommandContext(ctx, qemuBinary, makeCommandLine(firmwarePath)...)
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("qemu start: %w", err)
	}
	err := cmd.Wait()
	if err != nil {
		return fmt.Errorf("qemu wait: %w", err)
	}
	return nil
}

func makeCommandLine(filename string) []string {
	var ext string
	if strings.HasSuffix(filename, ".qcow2") {
		ext = "qcow2"
	} else {
		ext = "raw"
	}
	options := []string{
		"-drive", fmt.Sprintf("file=%s,format=%s", filename, ext),
		"-m", "512",
		"-netdev", getNativeNetworking("net0"),
		"-device", "virtio-net,netdev=net0,mac=" + generateMac(filename),
		"-nographic",
	}
	if runtime.GOOS == "linux" {
		options = append(options, "-enable-kvm")
	}
	return options
}

// getNativeNetworking returns the correct -netdev string for the current OS
func getNativeNetworking(id string) string {
	switch runtime.GOOS {
	case "darwin":
		return fmt.Sprintf("vmnet-shared,id=%s", id)
	case "linux":
		return fmt.Sprintf("tap,id=%s,ifname=tap0,br=br0,script=no", id)
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
	return fmt.Sprintf("%s:%02X:%02X:%02X:%02X", prefix, b1, b2, b3, b4)
}

package main

import (
	"context"
	"fmt"
	"github.com/perbu/qemu-wrapper/tuntap"
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

type Runner struct {
	tt         *tuntap.Manager
	options    []string
	firmware   string
	mac        string
	telnetPort uint16
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM)
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
	runner := &Runner{
		tt:       tuntap.New(),
		firmware: firmwarePath,
	}
	runner.generateMac()
	runner.allocatePort()
	runner.makeCommandLine()
	cmd := exec.CommandContext(ctx, qemuBinary, runner.options...)
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	for _, opt := range runner.options {
		fmt.Printf(" - %s\n", opt)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("qemu start: %w", err)
	}
	err := cmd.Wait()
	if err != nil {
		return fmt.Errorf("qemu wait: %w", err)
	}
	err = runner.tt.DeleteTaps()
	if err != nil {
		return fmt.Errorf("delete taps: %w", err)
	}
	return nil
}

func (r *Runner) makeCommandLine() {
	var ext string
	if strings.HasSuffix(r.firmware, ".qcow2") {
		ext = "qcow2"
	} else {
		ext = "raw"
	}
	options := []string{
		"-drive", fmt.Sprintf("file=%s,format=%s", r.firmware, ext),
		"-m", "512",
		"-machine", "q35",
		"-netdev", r.getNativeNetworking("net0"),
		"-device", fmt.Sprintf("virtio-net-pci,netdev=net0,mac=%s", r.mac),
		"-nographic",
		"-serial", fmt.Sprintf("telnet:localhost:%d,server,nowait", r.telnetPort),
	}
	if runtime.GOOS == "linux" {
		options = append(options, "-enable-kvm")
	}
	r.options = options
}

// getNativeNetworking returns the correct -netdev string for the current OS
func (r *Runner) getNativeNetworking(id string) string {
	switch runtime.GOOS {
	case "darwin":
		return fmt.Sprintf("vmnet-shared,id=%s", id)
	case "linux":
		// use a tap device.
		tapName := generateTapName(r.firmware)
		r.tt.SetSudo(true)
		err := r.tt.Load()
		if err != nil {
			panic(err)
		}
		err = r.tt.CreateTap(tapName)
		if err != nil {
			panic(err)
		}
		// add the tap to the bridge
		err = r.tt.AddTapToBridge(tapName, "br0")
		if err != nil {
			panic(err)
		}
		return fmt.Sprintf("tap,id=%s,ifname=%s,br=br0,script=no", id, tapName)
	default:
		panic("unsupported OS")
	}
}

func (r *Runner) allocatePort() {
	input := []string{r.firmware}
	input = append(input, os.Getenv("USER")) // add username to the input
	joined := strings.Join(input, "")
	if len(joined) == 0 {
		panic("no input to generate port")
	}
	// use crc32 to generate a 32-bit hash
	hash := crc32.ChecksumIEEE([]byte(joined))
	// use 1024 as the base port
	port := uint16(1024 + hash%100)
	fmt.Printf("Alloceated port %d on localhost for telnet to console\n", port)
	r.telnetPort = port
}

func (r *Runner) generateMac() {
	input := []string{r.firmware}
	input = append(input, os.Getenv("USER")) // add username to the input
	joined := strings.Join(input, "")
	if len(joined) == 0 {
		panic("no input to generate mac")
	}
	// use crc32 to generate a 32-bit hash
	hash := crc32.ChecksumIEEE([]byte(joined))
	// use qemu prefix 52:54
	r.mac = macFromInt("52:54", hash)
	fmt.Printf("Generated mac %s for the virtual machine\n", r.mac)
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

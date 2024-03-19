package tuntap

import (
	_ "embed"
	"fmt"
	"log"
	"testing"
)

//go:embed testdata/tap-output.txt
var tap_list_output []byte

//go:embed testdata/bridge-listing.txt
var bridge_list_output []byte

//go:embed testdata/taps-on-bridge-br0.txt
var bridge_tap_list_output_br0 []byte

//go:embed testdata/taps-on-bridge-br1.txt
var bridge_tap_list_output_br1 []byte

//go:embed testdata/tap-with-tailscale.txt
var tap_with_tailscale []byte

type mockExecutor struct {
	noOutput bool
}

func (e *mockExecutor) Run(path string, args ...string) ([]byte, error) {
	if e.noOutput {
		return nil, nil
	}
	if path == "ip" && args[0] == "link" && args[1] == "show" && args[2] == "type" && args[3] == "tun" {
		log.Println("mockExecutor listing taps:", path, args)
		return tap_list_output, nil
	}
	if path == "ip" && args[0] == "link" && args[1] == "show" && args[2] == "type" && args[3] == "bridge" {
		log.Println("mockExecutor listing bridges:", path, args)
		return bridge_list_output, nil
	}
	if path == "ip" && args[0] == "link" && args[1] == "show" && args[2] == "master" && args[4] == "type" && args[5] == "tun" && args[3] == "br0" {
		log.Println("mockExecutor listing taps on br0:", path, args)
		return bridge_tap_list_output_br0, nil
	}
	if path == "ip" && args[0] == "link" && args[1] == "show" && args[2] == "master" && args[4] == "type" && args[5] == "tun" && args[3] == "br1" {
		log.Println("mockExecutor listing taps on br1:", path, args)
		return bridge_tap_list_output_br1, nil
	}
	if path == "ip" && args[0] == "tuntap" && args[1] == "del" && args[2] == "dev" && args[4] == "mode" && args[5] == "tap" {
		log.Println("mockExecutor deleting tap:", args[3])
		return nil, nil
	}
	log.Println("mockExecutor unknown command:", path, args)
	panic("unknown command")
}

func TestManager_Load(t *testing.T) {
	mock := &mockExecutor{}

	m := New()
	m.SetSudo(false)
	m.commander = mock
	err := m.Load()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if len(m.taps) != 4 {
		t.Errorf("expected 4 taps, got %d", len(m.taps))
	}
	if len(m.bridges) != 2 {
		t.Errorf("expected 2 bridges, got %d", len(m.bridges))
	}
	if len(m.bridges["br0"].ifaces) != 2 {
		t.Errorf("expected 2 interfaces on br0, got %d", len(m.bridges["br0"].ifaces))
	}
	if len(m.bridges["br1"].ifaces) != 2 {
		t.Errorf("expected 2 interfaces on br1, got %d", len(m.bridges["br1"].ifaces))
	}
	for _, tap := range m.taps {
		if tap.bridge == nil {
			t.Errorf("expected tap %s to have a bridge", tap.name)
		}
	}
	if m.taps["tap0"].bridge.name != "br0" {
		t.Errorf("expected tap0 to be on br0, got %s", m.taps["tap0"].bridge.name)
	}
	if m.taps["tap1"].bridge.name != "br0" {
		t.Errorf("expected tap1 to be on br0, got %s", m.taps["tap1"].bridge.name)
	}
	if m.taps["tap2"].bridge.name != "br1" {
		t.Errorf("expected tap2 to be on br1, got %s", m.taps["tap2"].bridge.name)
	}
	if m.taps["tap3"].bridge.name != "br1" {
		t.Errorf("expected tap3 to be on br1, got %s", m.taps["tap3"].bridge.name)
	}
	if !(m.bridges["br0"].ifaces[0].name == "tap0" || m.bridges["br0"].ifaces[1].name == "tap0") {
		t.Errorf("expected br0 to have tap0, couldn't find it")
	}
	if !(m.bridges["br0"].ifaces[0].name == "tap1" || m.bridges["br0"].ifaces[1].name == "tap1") {
		t.Errorf("expected br0 to have tap1, couldn't find it")
	}
	if !(m.bridges["br1"].ifaces[0].name == "tap2" || m.bridges["br1"].ifaces[1].name == "tap2") {
		t.Errorf("expected br0 to have tap1, couldn't find it")
	}
	if !(m.bridges["br1"].ifaces[0].name == "tap3" || m.bridges["br1"].ifaces[1].name == "tap3") {
		t.Errorf("expected br0 to have tap1, couldn't find it")
	}
}

func TestManager_DeleteTaps(t *testing.T) {
	mock := &mockExecutor{}
	m := New()
	m.SetSudo(false)
	m.commander = mock
	err := m.Load()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	// mark all the taps as "mine"
	for _, tap := range m.taps {
		tap.mine = true
	}
	err = m.DeleteTaps()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if len(m.taps) != 0 {
		t.Errorf("expected 0 taps, got %d", len(m.taps))
	}
	if len(m.bridges) != 2 {
		t.Errorf("expected 2 bridges, got %d", len(m.bridges))
	}
	// iterate over bridges and check that they have no interfaces
	for _, bridge := range m.bridges {
		if len(bridge.ifaces) != 0 {
			t.Errorf("expected bridge %s to have no interfaces, got %d", bridge.name, len(bridge.ifaces))
		}
		log.Println("bridge", bridge.String())
	}
}

func TestManager_NoTaps(t *testing.T) {
	mock := &mockExecutor{
		noOutput: true,
	}
	m := New()
	m.SetSudo(false)
	m.commander = mock
	err := m.Load()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

}

func Test_makeRandomMac(t *testing.T) {
	for i := 0; i < 10; i++ {
		fmt.Println(makeRandomMac())
	}
}

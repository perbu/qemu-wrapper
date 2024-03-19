package tuntap

import (
	_ "embed"
	"fmt"
	"testing"
)

//go:embed testdata/tap-output.txt
var output []byte

func Test_parseTapListing(t *testing.T) {
	ifaces := parseTaps(output)
	if len(ifaces) != 4 {
		t.Errorf("expected 4 tap interfaces, got %d", len(ifaces))
	}
	for _, iface := range ifaces {
		fmt.Println("Interface:", iface.name, "mac:", iface.mac)
	}

}

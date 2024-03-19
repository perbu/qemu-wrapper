package tuntap

import (
	"testing"
)

func TestManager_parseTap(t *testing.T) {
	taps := parseTaps(tap_with_tailscale)
	if len(taps) != 2 {
		t.Fatalf("expected 2 taps, got %d", len(taps))
	}
	if taps[0].name != "tap0" {
		t.Errorf("expected name tap0, got %s", taps[1].name)
	}
	if taps[1].name != "tap1" {
		t.Errorf("expected name tap1, got %s", taps[2].name)
	}
	if taps[0].mac != "12:34:56:67:89:ab" {
		t.Errorf("expected mac 12:34:56:67:89:ab, got %s", taps[1].mac)
	}
	if taps[1].mac != "12:34:56:67:89:ad" {
		t.Errorf("expected mac 12:34:56:67:89:ad, got %s", taps[2].mac)
	}
}

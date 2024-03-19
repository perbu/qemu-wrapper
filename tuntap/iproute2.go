package tuntap

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

const forceMac = false

// split splits the output of ip link into a list of interfaces, one string per interface.
func split(input []byte) []string {
	// Split the input by newlines
	lines := strings.Split(string(input), "\n")

	var interfaces []string
	var tempInterface []string

	r := regexp.MustCompile(`^\d+: [\w\d]+:`)
	// Loop through the lines
	for _, line := range lines {

		// Check if line is the start of an interface description
		matched := r.MatchString(line)
		if matched {
			// If there is a collected interface, append it to the list
			if len(tempInterface) > 0 {
				interfaces = append(interfaces, strings.Join(tempInterface, " "))
				tempInterface = nil
			}
		}
		// Collect the line as part of the current interface
		tempInterface = append(tempInterface, line)
	}
	// Append the last collected interface
	if len(tempInterface) > 0 {
		interfaces = append(interfaces, strings.Join(tempInterface, " "))
	}
	return interfaces
}

// parseTapListing parses the output of `ip tuntap list` and returns a list of tap taps that are found.
// Example output:
// 30: tap0: <NO-CARRIER,BROADCAST,MULTICAST,UP> mtu 1500 qdisc pfifo_fast state DOWN mode DEFAULT group default qlen 1000
//
//	link/ether 12:85:f7:b0:07:54 brd ff:ff:ff:ff:ff:ff
func parseTaps(listing []byte) []tap {
	var taps []tap
	// make a regular expression that will get the names and macs of the tap taps:
	// r := regexp.MustCompile(`(?is)(\d+): (\w+):.*?link/(:?ether|none) *((?:(?:[0-9a-f]{2}:){5}[0-9a-f]{2})|none)`)
	// make a regex that will split the output into sections for each tap:

	ifs := split(listing)
	r := regexp.MustCompile(`(?is)(\d+): (\w+):.*?link/ether ((?:[0-9a-f]{2}:){5}[0-9a-f]{2})`)
	for _, iface := range ifs {
		// fmt.Println(iface)
		match := r.FindStringSubmatch(iface)
		if len(match) == 0 {
			continue
		}
		name := match[2]
		mac := match[3]
		// force the mac to lower case:
		mac = strings.ToLower(mac) //nolint:staticcheck
		taps = append(taps, tap{name: name, mac: match[3]})
	}

	return taps
}
func parseBridges(listing []byte) []bridge {
	var bridges []bridge
	// make a regular expression that will get the names and macs of the tap taps:
	r := regexp.MustCompile(`(?is)(\d+): ([\w-]+):.*?link/ether ((?:[0-9a-f]{2}:){5}[0-9a-f]{2})`)
	matches := r.FindAllStringSubmatch(string(listing), -1)

	for _, match := range matches {
		name := match[2]
		mac := match[3]
		// force the mac to lower case:
		mac = strings.ToLower(mac) //nolint:staticcheck
		bridges = append(bridges, bridge{name: name, mac: match[3]})
	}
	return bridges
}

// createBridge creates a bridge with the given name using the ip command.
func (m *Manager) createBridge(name string) error {
	var path string
	var args []string
	switch m.useSudo {
	case true:
		path = "sudo"
		args = []string{"ip", "link", "add", "name", name, "type", "bridge"}
	case false:
		path = "ip"
		args = []string{"link", "add", "name", name, "type", "bridge"}
	}
	_, err := m.commander.Run(path, args...)
	if err != nil {
		return fmt.Errorf("creating bridge: %w", err)
	}
	// set link state up:
	switch m.useSudo {
	case true:
		path = "sudo"
		args = []string{"ip", "link", "set", "dev", name, "up"}
	case false:
		path = "ip"
		args = []string{"link", "set", "dev", name, "up"}
	}
	_, err = m.commander.Run(path, args...)
	if err != nil {
		return fmt.Errorf("setting link state up on bridge: %w", err)
	}
	return nil
}

// createTap creates a tap interface with the given name and mac address
// using the ip command. it sets the interface to the UP state.
func (m *Manager) createTap(name, mac string) error {
	var path string
	var args []string

	user, ok := os.LookupEnv("USER")
	if !ok {
		return fmt.Errorf("USER environment variable not set")
	}

	switch m.useSudo {
	case true:
		path = "sudo"
		args = []string{"ip", "tuntap", "add", "dev", name, "mode", "tap"}
	case false:
		path = "ip"
		args = []string{"tuntap", "add", "dev", name, "mode", "tap"}
	}
	if user != "" {
		args = append(args, "user", user)
	}

	_, err := m.commander.Run(path, args...)
	if err != nil {
		return fmt.Errorf("creating tap interface: %w", err)
	}
	if forceMac {
		// set the mac address on the newly created tap interface:
		switch m.useSudo {
		case true:
			path = "sudo"
			args = []string{"ip", "link", "set", "dev", name, "address", mac}
		case false:
			path = "ip"
			args = []string{"link", "set", "dev", name, "address", mac}
		}
		_, err = m.commander.Run(path, args...)
		if err != nil {
			return fmt.Errorf("setting mac address on tap interface: %w", err)
		}
	}
	// set link state up:
	switch m.useSudo {
	case true:
		path = "sudo"
		args = []string{"ip", "link", "set", "dev", name, "up"}
	case false:
		path = "ip"
		args = []string{"link", "set", "dev", name, "up"}
	}
	_, err = m.commander.Run(path, args...)
	if err != nil {
		return fmt.Errorf("setting link state up on tap interface: %w", err)
	}
	if err != nil {
		return fmt.Errorf("setting mac address on tap interface: %w", err)
	}
	return nil
}

// delete tap will delete the tap interface with the given name.
func (m *Manager) deleteTap(name string) error {
	var path string
	var args []string
	switch m.useSudo {
	case true:
		path = "sudo"
		args = []string{"ip", "tuntap", "del", "dev", name, "mode", "tap"}
	case false:
		path = "ip"
		args = []string{"tuntap", "del", "dev", name, "mode", "tap"}
	}
	_, err := m.commander.Run(path, args...)
	if err != nil {
		return fmt.Errorf("deleting tap interfaces: %w", err)
	}
	return nil
}

func (m *Manager) addTapToBridge(useSudo bool, name, bridge string) error {
	var path string
	var args []string
	switch useSudo {
	case true:
		path = "sudo"
		args = []string{"ip", "link", "set", name, "master", bridge}
	case false:
		path = "ip"
		args = []string{"link", "set", name, "master", bridge}
	}
	_, err := m.commander.Run(path, args...)
	if err != nil {
		return fmt.Errorf("adding tap to bridge: %w", err)
	}
	return nil
}

func (m *Manager) listTaps() ([]tap, error) {
	var path string
	var args []string
	switch m.useSudo {
	case true:
		path = "sudo"
		args = []string{"ip", "link", "show", "type", "tun"}
	case false:
		path = "ip"
		args = []string{"link", "show", "type", "tun"}
	}
	out, err := m.commander.Run(path, args...)
	if err != nil {
		return nil, fmt.Errorf("listing taps: %w", err)
	}
	taps := parseTaps(out)
	return taps, nil
}

func (m *Manager) listBridges() ([]bridge, error) {
	var path string
	var args []string
	switch m.useSudo {
	case true:
		path = "sudo"
		args = []string{"ip", "link", "show", "type", "bridge"}
	case false:
		path = "ip"
		args = []string{"link", "show", "type", "bridge"}
	}
	out, err := m.commander.Run(path, args...)
	if err != nil {
		return nil, fmt.Errorf("listing bridges: %w", err)
	}
	bridges := parseBridges(out)
	return bridges, nil
}

func (m *Manager) listTapsOnBridge(br string) ([]tap, error) {
	var path string
	var args []string
	switch m.useSudo {
	case true:
		path = "sudo"
		args = []string{"ip", "link", "show", "master", br, "type", "tun"}
	case false:
		path = "ip"
		args = []string{"link", "show", "master", br, "type", "tun"}
	}
	out, err := m.commander.Run(path, args...)
	if err != nil {
		return nil, fmt.Errorf("listing taps: %w", err)
	}
	taps := parseTaps(out)
	return taps, nil
}

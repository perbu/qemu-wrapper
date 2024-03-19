package tuntap

import "fmt"

type bridge struct {
	name   string
	ifaces []*tap
	mac    string
}

func (br *bridge) addTap(iface *tap) error {
	// check if iface is already in bridge
	for _, i := range br.ifaces {
		if i == iface {
			return fmt.Errorf("iface %s already in bridge %s", iface.name, br.name)
		}
	}
	br.ifaces = append(br.ifaces, iface)
	return nil
}

func (br *bridge) removeTap(iface *tap) error {
	// check if iface is already in bridge
	for i, t := range br.ifaces {
		if t == iface {
			br.ifaces = append(br.ifaces[:i], br.ifaces[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("iface %s not in bridge %s", iface.name, br.name)
}

func (br *bridge) String() string {
	ifaceList := ""
	for _, i := range br.ifaces {
		ifaceList += i.String() + " "
	}
	return fmt.Sprintf("[%s %s] taps: %s", br.name, br.mac, ifaceList)
}

package tuntap

import "fmt"

type tap struct {
	name   string
	mac    string
	bridge *bridge
	mine   bool // true if this tap was created by us
}

func (t *tap) String() string {
	return fmt.Sprintf("[%s %s]", t.name, t.mac)
}

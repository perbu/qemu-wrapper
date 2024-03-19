package tuntap

import (
	"fmt"
	"os"
	"sync"
)

type tapMap map[string]*tap
type bridgeMap map[string]*bridge

type Manager struct {
	mu        sync.Mutex
	taps      tapMap
	bridges   bridgeMap
	useSudo   bool
	commander Executor
}

func New() *Manager {
	exe := newExecutor()

	return &Manager{
		taps:      make(tapMap),
		bridges:   make(bridgeMap),
		commander: exe,
		mu:        sync.Mutex{},
	}
}

func (m *Manager) OverrideCommander(e Executor) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.commander = e
}

func (m *Manager) SetSudo(useSudo bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.useSudo = useSudo
}

func (m *bridgeMap) String() string {
	s := ""
	for _, b := range *m {
		s += b.String() + "\n"
	}
	return s
}
func (m *Manager) String() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.bridges.String()
}

// Load loads existing tap taps and bridges

func (m *Manager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	// clear out the existing state
	m.taps = make(map[string]*tap)
	m.bridges = make(map[string]*bridge)
	// list the taps
	taps, err := m.listTaps()
	if err != nil {
		return fmt.Errorf("listTaps: %w", err)
	}
	for _, t := range taps {
		m.taps[t.name] = &tap{name: t.name, mac: t.mac}
	}
	bridges, err := m.listBridges()
	if err != nil {
		return fmt.Errorf("listBridges: %w", err)
	}
	// register the bridges
	for _, br := range bridges {

		mybr := &bridge{
			name:   br.name,
			mac:    br.mac,
			ifaces: make([]*tap, 0),
		}
		// now list the taps on the bridge
		brtaps, err := m.listTapsOnBridge(br.name)
		if err != nil {
			return fmt.Errorf("listTapsOnBridge: %w", err)
		}
		// add the taps to the bridge
		for _, t := range brtaps {
			brtap, ok := m.taps[t.name] // get the tap from the manager
			if !ok {
				return fmt.Errorf("tap %s not found", t.name)
			}
			brtap.bridge = mybr
			mybr.ifaces = append(mybr.ifaces, brtap)
		}
		m.bridges[br.name] = mybr
	}
	return nil
}

func (m *Manager) CreateTap(tapName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.taps[tapName]; ok {
		return fmt.Errorf("tap device %s already exists", tapName)
	}
	userName, ok := os.LookupEnv("USER")
	if !ok {
		userName = "unknown"
	}

	mac := makeRandomMac(userName + tapName)
	err := m.createTap(tapName, mac)
	if err != nil {
		// cleanup
		_ = m.deleteTap(tapName)
		return fmt.Errorf("creating tap device: %w", err)
	}
	m.taps[tapName] = &tap{name: tapName, mac: mac, mine: true}
	return nil
}

func (m *Manager) GetMac(name string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.taps[name]
	if !ok {
		return "", fmt.Errorf("tap device %s does not exist", name)
	}
	return t.mac, nil
}

// DeleteTaps deletes all the taps created by vmm and updates the data structures.
func (m *Manager) DeleteTaps() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	// first we iterate over all the bridges and remove the taps from them
	for _, br := range m.bridges {
		for _, t := range br.ifaces {
			if !t.mine {
				continue
			}
			err := br.removeTap(t)
			if err != nil {
				return fmt.Errorf("removeTap: %w", err)
			}
		}
	}
	// now we delete all the taps from the manager and the underlying system:
	for _, t := range m.taps {
		if !t.mine {
			continue
		}
		err := m.deleteTap(t.name)
		if err != nil {
			return fmt.Errorf("deleteTap: %w", err)
		}
		// now remove the tap from the manager
		delete(m.taps, t.name)
	}
	return nil
}

func (m *Manager) AddTapToBridge(name, bridge string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.taps[name]
	if !ok {
		return fmt.Errorf("tap device %s does not exist", name)
	}
	br, ok := m.bridges[bridge]
	if !ok {
		return fmt.Errorf("bridge %s does not exist", bridge)
	}
	err := br.addTap(t)
	if err != nil {
		return fmt.Errorf("addTapToBridge: %w", err)
	}
	err = m.addTapToBridge(m.useSudo, name, bridge)
	if err != nil {
		return fmt.Errorf("addTapToBridge: %w", err)
	}
	m.taps[name].bridge = br
	return nil
}

func (m *Manager) HasTap(tap string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.taps[tap]
	return ok
}

func (m *Manager) HasBridge(bridge string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.bridges[bridge]
	return ok
}

func (m *Manager) CreateBridge(brname string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.bridges[brname]; ok {
		return fmt.Errorf("bridge %s already exists", brname)
	}
	err := m.createBridge(brname)
	if err != nil {
		return fmt.Errorf("createBridge: %w", err)
	}
	m.bridges[brname] = &bridge{name: brname, ifaces: make([]*tap, 0)}
	return nil
}

func (m *Manager) BridgeHasTap(bridge, tap string) bool {
	// check if the tap is on the bridge
	m.mu.Lock()
	defer m.mu.Unlock()
	br, ok := m.bridges[bridge]
	if !ok {
		return false
	}
	for _, t := range br.ifaces {
		if t.name == tap {
			return true
		}
	}
	return false
}

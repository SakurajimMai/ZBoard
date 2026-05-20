package agent

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

// portHoppingConfig is the `_zboard_port_hopping` block embedded in the
// runtime config JSON by the control plane for Hysteria2 nodes with port_range.
type portHoppingConfig struct {
	Enabled      bool     `json:"enabled"`
	ListenPort   int      `json:"listen_port"`
	PortRange    string   `json:"port_range"`
	SetupCmds    []string `json:"setup_cmds"`
	TeardownCmds []string `json:"teardown_cmds"`
}

// applyPortHopping parses the runtime config for `_zboard_port_hopping` and
// executes the iptables setup commands. It first tears down any existing rules
// (idempotent) then applies the new ones.
func applyPortHopping(configJSON []byte) {
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(configJSON, &doc); err != nil {
		return
	}
	raw, ok := doc["_zboard_port_hopping"]
	if !ok {
		return
	}
	var ph portHoppingConfig
	if err := json.Unmarshal(raw, &ph); err != nil || !ph.Enabled {
		return
	}
	if ph.PortRange == "" || len(ph.SetupCmds) == 0 {
		return
	}

	log.Printf("port-hopping: setting up %s -> :%d", ph.PortRange, ph.ListenPort)

	// Teardown first (ignore errors — rules may not exist yet).
	for _, cmd := range ph.TeardownCmds {
		runShell(cmd, true)
	}
	// Setup.
	for _, cmd := range ph.SetupCmds {
		if err := runShell(cmd, false); err != nil {
			log.Printf("port-hopping: setup cmd failed: %s -> %v", cmd, err)
		}
	}
}

// teardownPortHopping removes iptables rules. Called before applying a new
// config (in case port range changed) and on agent shutdown.
func teardownPortHopping(configJSON []byte) {
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(configJSON, &doc); err != nil {
		return
	}
	raw, ok := doc["_zboard_port_hopping"]
	if !ok {
		return
	}
	var ph portHoppingConfig
	if err := json.Unmarshal(raw, &ph); err != nil || !ph.Enabled {
		return
	}
	for _, cmd := range ph.TeardownCmds {
		runShell(cmd, true)
	}
}

func runShell(cmd string, ignoreErr bool) error {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return nil
	}
	c := exec.Command(parts[0], parts[1:]...)
	out, err := c.CombinedOutput()
	if err != nil && !ignoreErr {
		return fmt.Errorf("%s: %s", err, string(out))
	}
	return nil
}

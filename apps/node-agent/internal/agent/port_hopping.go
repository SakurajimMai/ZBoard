package agent

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

// portHoppingConfig is the `port_hopping` block from the sync_config task
// payload (NOT from the runtime config JSON — sing-box rejects unknown fields).
type portHoppingConfig struct {
	Enabled      bool     `json:"enabled"`
	ListenPort   int      `json:"listen_port"`
	PortRange    string   `json:"port_range"`
	SetupCmds    []string `json:"setup_cmds"`
	TeardownCmds []string `json:"teardown_cmds"`
}

// applyPortHopping reads the `port_hopping` block from the task payload JSON
// and executes the iptables setup commands. Returns an error if any setup
// command fails so the caller can report the task as failed.
func applyPortHopping(taskPayload []byte) error {
	ph, err := parsePortHopping(taskPayload)
	if err != nil || ph == nil {
		return nil // no port hopping configured — not an error
	}

	log.Printf("port-hopping: setting up %s -> :%d", ph.PortRange, ph.ListenPort)

	// Teardown first (ignore errors — rules may not exist yet).
	for _, cmd := range ph.TeardownCmds {
		runShell(cmd, true)
	}
	// Setup — any failure is fatal.
	for _, cmd := range ph.SetupCmds {
		if err := runShell(cmd, false); err != nil {
			return fmt.Errorf("port-hopping setup failed: %s -> %w", cmd, err)
		}
	}
	return nil
}

// teardownPortHopping removes iptables rules. Called before applying a new
// config (in case port range changed) and on agent shutdown.
func teardownPortHopping(taskPayload []byte) {
	ph, err := parsePortHopping(taskPayload)
	if err != nil || ph == nil {
		return
	}
	log.Printf("port-hopping: tearing down %s", ph.PortRange)
	for _, cmd := range ph.TeardownCmds {
		runShell(cmd, true)
	}
}

// parsePortHopping extracts the port_hopping config from a task payload JSON.
// Returns nil (no error) when the payload doesn't contain port_hopping.
func parsePortHopping(payload []byte) (*portHoppingConfig, error) {
	var doc struct {
		PortHopping *portHoppingConfig `json:"port_hopping"`
	}
	if err := json.Unmarshal(payload, &doc); err != nil {
		return nil, err
	}
	if doc.PortHopping == nil || !doc.PortHopping.Enabled {
		return nil, nil
	}
	if doc.PortHopping.PortRange == "" || len(doc.PortHopping.SetupCmds) == 0 {
		return nil, nil
	}
	return doc.PortHopping, nil
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

// Package firewall abstracts the host firewall, dispatching to ufw on the
// Debian family and firewalld on the RHEL family.
package firewall

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/jud3ns0hn/digitarlopanel/backend/internal/pkg/runner"
)

// Backend identifies which firewall tool is in use.
type Backend string

const (
	BackendUFW       Backend = "ufw"
	BackendFirewalld Backend = "firewalld"
	BackendNone      Backend = "none"
)

// Manager controls the detected host firewall.
type Manager struct {
	backend Backend
	runner  runner.Runner
}

// Rule is a normalized firewall rule.
type Rule struct {
	Port  int    `json:"port"`
	Proto string `json:"proto"` // tcp | udp
	Raw   string `json:"raw"`
}

// New detects the available firewall backend.
func New(r runner.Runner) *Manager {
	b := BackendNone
	if _, err := exec.LookPath("ufw"); err == nil {
		b = BackendUFW
	} else if _, err := exec.LookPath("firewall-cmd"); err == nil {
		b = BackendFirewalld
	}
	return &Manager{backend: b, runner: r}
}

// Backend returns the detected backend name.
func (m *Manager) Backend() Backend { return m.backend }

// Status returns the raw status output of the firewall tool.
func (m *Manager) Status(ctx context.Context) (string, error) {
	switch m.backend {
	case BackendUFW:
		return m.out(ctx, "ufw", "status", "verbose")
	case BackendFirewalld:
		return m.out(ctx, "firewall-cmd", "--state")
	default:
		return "", errNoBackend
	}
}

// List returns the currently allowed ports.
func (m *Manager) List(ctx context.Context) ([]Rule, error) {
	switch m.backend {
	case BackendUFW:
		return m.listUFW(ctx)
	case BackendFirewalld:
		return m.listFirewalld(ctx)
	default:
		return nil, errNoBackend
	}
}

// Allow opens a port for the given protocol.
func (m *Manager) Allow(ctx context.Context, port int, proto string) (string, error) {
	if err := validate(port, proto); err != nil {
		return "", err
	}
	spec := fmt.Sprintf("%d/%s", port, proto)
	switch m.backend {
	case BackendUFW:
		return m.out(ctx, "ufw", "allow", spec)
	case BackendFirewalld:
		if _, err := m.out(ctx, "firewall-cmd", "--permanent", "--add-port="+spec); err != nil {
			return "", err
		}
		return m.out(ctx, "firewall-cmd", "--reload")
	default:
		return "", errNoBackend
	}
}

// Deny removes a previously allowed port.
func (m *Manager) Deny(ctx context.Context, port int, proto string) (string, error) {
	if err := validate(port, proto); err != nil {
		return "", err
	}
	spec := fmt.Sprintf("%d/%s", port, proto)
	switch m.backend {
	case BackendUFW:
		return m.out(ctx, "ufw", "delete", "allow", spec)
	case BackendFirewalld:
		if _, err := m.out(ctx, "firewall-cmd", "--permanent", "--remove-port="+spec); err != nil {
			return "", err
		}
		return m.out(ctx, "firewall-cmd", "--reload")
	default:
		return "", errNoBackend
	}
}

func (m *Manager) listUFW(ctx context.Context) ([]Rule, error) {
	out, err := m.out(ctx, "ufw", "status")
	if err != nil {
		return nil, err
	}
	var rules []Rule
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || !strings.Contains(fields[0], "/") {
			continue
		}
		port, proto, ok := splitPortProto(fields[0])
		if !ok {
			continue
		}
		rules = append(rules, Rule{Port: port, Proto: proto, Raw: strings.TrimSpace(line)})
	}
	return rules, nil
}

func (m *Manager) listFirewalld(ctx context.Context) ([]Rule, error) {
	out, err := m.out(ctx, "firewall-cmd", "--list-ports")
	if err != nil {
		return nil, err
	}
	var rules []Rule
	for _, tok := range strings.Fields(out) {
		port, proto, ok := splitPortProto(tok)
		if !ok {
			continue
		}
		rules = append(rules, Rule{Port: port, Proto: proto, Raw: tok})
	}
	return rules, nil
}

func splitPortProto(s string) (int, string, bool) {
	portStr, proto, ok := strings.Cut(s, "/")
	if !ok {
		return 0, "", false
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return 0, "", false
	}
	proto = strings.ToLower(proto)
	if proto != "tcp" && proto != "udp" {
		return 0, "", false
	}
	return port, proto, true
}

func validate(port int, proto string) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("invalid port: %d", port)
	}
	if proto != "tcp" && proto != "udp" {
		return fmt.Errorf("invalid protocol: %s", proto)
	}
	return nil
}

func (m *Manager) out(ctx context.Context, name string, args ...string) (string, error) {
	res, err := m.runner.Run(ctx, name, args...)
	out := res.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("%s %v: %w", name, args, err)
	}
	return out, nil
}

var errNoBackend = fmt.Errorf("no supported firewall backend found (ufw or firewalld)")

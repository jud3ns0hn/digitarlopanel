// Package pkgmgr provides a cross-distro package management interface that
// dispatches to apt-get (Debian family) or dnf/yum (RHEL family).
package pkgmgr

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/jud3ns0hn/digitarlopanel/backend/internal/pkg/osinfo"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/pkg/runner"
)

// Manager installs and removes OS packages for a given distribution family.
type Manager struct {
	family osinfo.Family
	tool   string // resolved package tool: apt-get, dnf or yum
	runner runner.Runner
}

// New builds a Manager for the given family using the provided runner.
// The RHEL tool resolves to dnf when available, otherwise yum.
func New(family osinfo.Family, r runner.Runner) *Manager {
	m := &Manager{family: family, runner: r}
	switch family {
	case osinfo.FamilyDebian:
		m.tool = "apt-get"
	case osinfo.FamilyRHEL:
		if _, err := exec.LookPath("dnf"); err == nil {
			m.tool = "dnf"
		} else {
			m.tool = "yum"
		}
	}
	return m
}

// Tool returns the resolved package manager binary name (for diagnostics).
func (m *Manager) Tool() string { return m.tool }

// Install installs the named package non-interactively.
func (m *Manager) Install(ctx context.Context, pkg string) (string, error) {
	switch m.family {
	case osinfo.FamilyDebian:
		return m.run(ctx, "apt-get", "install", "-y", pkg)
	case osinfo.FamilyRHEL:
		return m.run(ctx, m.tool, "install", "-y", pkg)
	default:
		return "", errUnsupported(m.family)
	}
}

// Remove uninstalls the named package non-interactively.
func (m *Manager) Remove(ctx context.Context, pkg string) (string, error) {
	switch m.family {
	case osinfo.FamilyDebian:
		return m.run(ctx, "apt-get", "remove", "-y", pkg)
	case osinfo.FamilyRHEL:
		return m.run(ctx, m.tool, "remove", "-y", pkg)
	default:
		return "", errUnsupported(m.family)
	}
}

// Update refreshes the package index (apt-get update / dnf makecache).
func (m *Manager) Update(ctx context.Context) (string, error) {
	switch m.family {
	case osinfo.FamilyDebian:
		return m.run(ctx, "apt-get", "update")
	case osinfo.FamilyRHEL:
		return m.run(ctx, m.tool, "makecache")
	default:
		return "", errUnsupported(m.family)
	}
}

// IsInstalled reports whether pkg is installed. It uses dpkg/rpm queries.
func (m *Manager) IsInstalled(ctx context.Context, pkg string) bool {
	switch m.family {
	case osinfo.FamilyDebian:
		res, err := m.runner.Run(ctx, "dpkg-query", "-W", "-f=${Status}", pkg)
		return err == nil && res.ExitCode == 0
	case osinfo.FamilyRHEL:
		res, err := m.runner.Run(ctx, "rpm", "-q", pkg)
		return err == nil && res.ExitCode == 0
	default:
		return false
	}
}

func (m *Manager) run(ctx context.Context, name string, args ...string) (string, error) {
	res, err := m.runner.Run(ctx, name, args...)
	out := res.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("%s %v: %w", name, args, err)
	}
	return out, nil
}

func errUnsupported(f osinfo.Family) error {
	return fmt.Errorf("unsupported distribution family: %s", f)
}

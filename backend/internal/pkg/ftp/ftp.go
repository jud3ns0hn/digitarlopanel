// Package ftp manages pure-ftpd virtual users via the pure-pw tool. Virtual
// users map to a single system user/group and authenticate against PureDB.
package ftp

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/jud3ns0hn/digitarlopanel/backend/internal/pkg/runner"
)

// Manager wraps pure-pw operations.
type Manager struct {
	runner    runner.InputRunner
	sysUser   string
	sysGroup  string
	available bool
}

// New returns a Manager. sysUser/sysGroup are the system account pure-ftpd
// virtual users map to.
func New(r runner.InputRunner, sysUser, sysGroup string) *Manager {
	_, err := exec.LookPath("pure-pw")
	return &Manager{runner: r, sysUser: sysUser, sysGroup: sysGroup, available: err == nil}
}

// Available reports whether pure-pw is installed.
func (m *Manager) Available() bool { return m.available }

// passwordStdin formats the two-line password input pure-pw expects.
func passwordStdin(password string) string {
	return password + "\n" + password + "\n"
}

// Create adds a virtual user with the given home directory and password, then
// rebuilds the PureDB database.
func (m *Manager) Create(ctx context.Context, username, home, password string) (string, error) {
	res, err := m.runner.RunInput(ctx, passwordStdin(password),
		"pure-pw", "useradd", username, "-u", m.sysUser, "-g", m.sysGroup, "-d", home, "-m")
	if err != nil {
		return res.CombinedOutput(), fmt.Errorf("pure-pw useradd: %w", err)
	}
	return res.CombinedOutput(), nil
}

// SetPassword changes a virtual user's password.
func (m *Manager) SetPassword(ctx context.Context, username, password string) (string, error) {
	res, err := m.runner.RunInput(ctx, passwordStdin(password), "pure-pw", "passwd", username, "-m")
	if err != nil {
		return res.CombinedOutput(), fmt.Errorf("pure-pw passwd: %w", err)
	}
	return res.CombinedOutput(), nil
}

// Delete removes a virtual user and rebuilds the database.
func (m *Manager) Delete(ctx context.Context, username string) (string, error) {
	res, err := m.runner.RunInput(ctx, "", "pure-pw", "userdel", username, "-m")
	if err != nil {
		return res.CombinedOutput(), fmt.Errorf("pure-pw userdel: %w", err)
	}
	return res.CombinedOutput(), nil
}

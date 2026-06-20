// Package service controls system daemons via systemd, which is present on all
// supported target distributions (Ubuntu/Debian and RHEL/CentOS/Rocky).
package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/jud3ns0hn/digitarlopanel/backend/internal/pkg/runner"
)

// Controller wraps systemctl operations.
type Controller struct {
	runner runner.Runner
}

// New returns a Controller using the given runner.
func New(r runner.Runner) *Controller {
	return &Controller{runner: r}
}

// Status describes a unit's current state.
type Status struct {
	Name    string `json:"name"`
	Active  bool   `json:"active"`  // is-active == "active"
	Enabled bool   `json:"enabled"` // is-enabled == "enabled"
	State   string `json:"state"`   // raw is-active output
}

func (c *Controller) systemctl(ctx context.Context, args ...string) (string, error) {
	res, err := c.runner.Run(ctx, "systemctl", args...)
	out := res.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("systemctl %v: %w", args, err)
	}
	return out, nil
}

// Start starts a unit.
func (c *Controller) Start(ctx context.Context, unit string) (string, error) {
	return c.systemctl(ctx, "start", unit)
}

// Stop stops a unit.
func (c *Controller) Stop(ctx context.Context, unit string) (string, error) {
	return c.systemctl(ctx, "stop", unit)
}

// Restart restarts a unit.
func (c *Controller) Restart(ctx context.Context, unit string) (string, error) {
	return c.systemctl(ctx, "restart", unit)
}

// Reload asks a unit to reload its configuration.
func (c *Controller) Reload(ctx context.Context, unit string) (string, error) {
	return c.systemctl(ctx, "reload", unit)
}

// Enable enables a unit at boot.
func (c *Controller) Enable(ctx context.Context, unit string) (string, error) {
	return c.systemctl(ctx, "enable", unit)
}

// Disable disables a unit at boot.
func (c *Controller) Disable(ctx context.Context, unit string) (string, error) {
	return c.systemctl(ctx, "disable", unit)
}

// Status reports whether a unit is active and enabled. systemctl returns a
// non-zero exit code for inactive units, which is expected and not an error.
func (c *Controller) Status(ctx context.Context, unit string) Status {
	active, _ := c.runner.Run(ctx, "systemctl", "is-active", unit)
	enabled, _ := c.runner.Run(ctx, "systemctl", "is-enabled", unit)
	state := strings.TrimSpace(active.Stdout)
	return Status{
		Name:    unit,
		Active:  state == "active",
		Enabled: strings.TrimSpace(enabled.Stdout) == "enabled",
		State:   state,
	}
}

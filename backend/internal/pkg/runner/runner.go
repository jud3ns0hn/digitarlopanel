// Package runner abstracts external command execution so that distro-specific
// logic can be unit tested with a mock instead of touching the host.
package runner

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"time"
)

// Result is the outcome of running a command.
type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// Runner executes external commands. Commands are always passed as an argument
// slice (never a shell string) to avoid injection.
type Runner interface {
	Run(ctx context.Context, name string, args ...string) (Result, error)
}

// Exec is the production Runner backed by os/exec.
type Exec struct {
	// Timeout caps the duration of a single command. Zero means 5 minutes.
	Timeout time.Duration
}

// Run executes name with args and captures stdout/stderr.
func (e Exec) Run(ctx context.Context, name string, args ...string) (Result, error) {
	timeout := e.Timeout
	if timeout == 0 {
		timeout = 5 * time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	res := Result{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}
	if cmd.ProcessState != nil {
		res.ExitCode = cmd.ProcessState.ExitCode()
	}
	return res, err
}

// CombinedOutput returns stdout and stderr concatenated, trimmed.
func (r Result) CombinedOutput() string {
	return strings.TrimSpace(r.Stdout + r.Stderr)
}

// Mock is a Runner that records calls and returns canned results, for tests.
type Mock struct {
	// Calls records each invocation as "name arg1 arg2 ...".
	Calls []string
	// Responder optionally maps an invocation to a result.
	Responder func(name string, args ...string) (Result, error)
}

// Run records the call and delegates to Responder if present.
func (m *Mock) Run(_ context.Context, name string, args ...string) (Result, error) {
	m.Calls = append(m.Calls, strings.TrimSpace(name+" "+strings.Join(args, " ")))
	if m.Responder != nil {
		return m.Responder(name, args...)
	}
	return Result{}, nil
}

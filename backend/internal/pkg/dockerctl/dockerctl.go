// Package dockerctl wraps the docker CLI to manage containers and images.
// It shells out to docker with structured Go-template output so the panel does
// not need the Docker SDK, keeping the binary small.
package dockerctl

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/jud3ns0hn/digitarlopanel/backend/internal/pkg/runner"
)

// Manager controls the local Docker engine via the docker CLI.
type Manager struct {
	runner    runner.Runner
	available bool
}

// New detects whether docker is installed.
func New(r runner.Runner) *Manager {
	_, err := exec.LookPath("docker")
	return &Manager{runner: r, available: err == nil}
}

// Available reports whether the docker CLI is present.
func (m *Manager) Available() bool { return m.available }

// Container is a normalized docker container summary.
type Container struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Image  string `json:"image"`
	State  string `json:"state"`
	Status string `json:"status"`
	Ports  string `json:"ports"`
}

// Image is a normalized docker image summary.
type Image struct {
	ID         string `json:"id"`
	Repository string `json:"repository"`
	Tag        string `json:"tag"`
	Size       string `json:"size"`
}

// validName guards container/image identifiers passed to the CLI.
func validName(s string) bool {
	if s == "" || len(s) > 128 {
		return false
	}
	for _, r := range s {
		if !(r == '-' || r == '_' || r == '.' || r == ':' || r == '/' || r == '@' ||
			(r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			return false
		}
	}
	return true
}

// Containers lists all containers (running and stopped).
func (m *Manager) Containers(ctx context.Context) ([]Container, error) {
	res, err := m.runner.Run(ctx, "docker", "ps", "-a", "--no-trunc",
		"--format", "{{json .}}")
	if err != nil {
		return nil, fmt.Errorf("docker ps: %s", res.CombinedOutput())
	}
	var out []Container
	for _, line := range strings.Split(strings.TrimSpace(res.Stdout), "\n") {
		if line == "" {
			continue
		}
		var raw struct {
			ID, Names, Image, State, Status, Ports string
		}
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue
		}
		out = append(out, Container{
			ID: raw.ID, Name: raw.Names, Image: raw.Image,
			State: raw.State, Status: raw.Status, Ports: raw.Ports,
		})
	}
	return out, nil
}

// Images lists local images.
func (m *Manager) Images(ctx context.Context) ([]Image, error) {
	res, err := m.runner.Run(ctx, "docker", "images", "--no-trunc", "--format", "{{json .}}")
	if err != nil {
		return nil, fmt.Errorf("docker images: %s", res.CombinedOutput())
	}
	var out []Image
	for _, line := range strings.Split(strings.TrimSpace(res.Stdout), "\n") {
		if line == "" {
			continue
		}
		var raw struct {
			ID, Repository, Tag, Size string
		}
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue
		}
		out = append(out, Image{ID: raw.ID, Repository: raw.Repository, Tag: raw.Tag, Size: raw.Size})
	}
	return out, nil
}

// ContainerAction runs a lifecycle action (start|stop|restart|remove) on a container.
func (m *Manager) ContainerAction(ctx context.Context, id, action string) (string, error) {
	if !validName(id) {
		return "", fmt.Errorf("invalid container id")
	}
	var args []string
	switch action {
	case "start":
		args = []string{"start", id}
	case "stop":
		args = []string{"stop", id}
	case "restart":
		args = []string{"restart", id}
	case "remove":
		args = []string{"rm", "-f", id}
	default:
		return "", fmt.Errorf("invalid action: %s", action)
	}
	res, err := m.runner.Run(ctx, "docker", args...)
	out := res.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("docker %s: %w", action, err)
	}
	return out, nil
}

// Pull downloads an image by reference.
func (m *Manager) Pull(ctx context.Context, image string) (string, error) {
	if !validName(image) {
		return "", fmt.Errorf("invalid image reference")
	}
	res, err := m.runner.Run(ctx, "docker", "pull", image)
	out := res.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("docker pull: %w", err)
	}
	return out, nil
}

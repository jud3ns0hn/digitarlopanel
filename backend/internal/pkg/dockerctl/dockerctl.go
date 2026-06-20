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

// Network is a normalized docker network summary.
type Network struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Driver string `json:"driver"`
	Scope  string `json:"scope"`
}

// Volume is a normalized docker volume summary.
type Volume struct {
	Name       string `json:"name"`
	Driver     string `json:"driver"`
	Mountpoint string `json:"mountpoint"`
}

// Networks lists docker networks.
func (m *Manager) Networks(ctx context.Context) ([]Network, error) {
	res, err := m.runner.Run(ctx, "docker", "network", "ls", "--no-trunc", "--format", "{{json .}}")
	if err != nil {
		return nil, fmt.Errorf("docker network ls: %s", res.CombinedOutput())
	}
	var out []Network
	for _, line := range strings.Split(strings.TrimSpace(res.Stdout), "\n") {
		if line == "" {
			continue
		}
		var raw struct{ ID, Name, Driver, Scope string }
		if json.Unmarshal([]byte(line), &raw) == nil {
			out = append(out, Network{ID: raw.ID, Name: raw.Name, Driver: raw.Driver, Scope: raw.Scope})
		}
	}
	return out, nil
}

// Volumes lists docker volumes.
func (m *Manager) Volumes(ctx context.Context) ([]Volume, error) {
	res, err := m.runner.Run(ctx, "docker", "volume", "ls", "--format", "{{json .}}")
	if err != nil {
		return nil, fmt.Errorf("docker volume ls: %s", res.CombinedOutput())
	}
	var out []Volume
	for _, line := range strings.Split(strings.TrimSpace(res.Stdout), "\n") {
		if line == "" {
			continue
		}
		var raw struct{ Name, Driver, Mountpoint string }
		if json.Unmarshal([]byte(line), &raw) == nil {
			out = append(out, Volume{Name: raw.Name, Driver: raw.Driver, Mountpoint: raw.Mountpoint})
		}
	}
	return out, nil
}

// ContainerLogs returns the last n log lines of a container.
func (m *Manager) ContainerLogs(ctx context.Context, id string, lines int) (string, error) {
	if !validName(id) {
		return "", fmt.Errorf("invalid container id")
	}
	res, err := m.runner.Run(ctx, "docker", "logs", "--tail", fmt.Sprint(lines), id)
	return res.CombinedOutput(), err
}

// ContainerStats returns a one-shot resource snapshot for a container.
func (m *Manager) ContainerStats(ctx context.Context, id string) (string, error) {
	if !validName(id) {
		return "", fmt.Errorf("invalid container id")
	}
	res, err := m.runner.Run(ctx, "docker", "stats", "--no-stream", "--format", "{{json .}}", id)
	return strings.TrimSpace(res.CombinedOutput()), err
}

// Prune removes unused docker data (containers, networks, images, build cache).
func (m *Manager) Prune(ctx context.Context) (string, error) {
	res, err := m.runner.Run(ctx, "docker", "system", "prune", "-f")
	out := res.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("docker prune: %w", err)
	}
	return out, nil
}

// ComposeUp deploys a compose project from dir in detached mode.
func (m *Manager) ComposeUp(ctx context.Context, dir, project string) (string, error) {
	if !validName(project) {
		return "", fmt.Errorf("invalid project name")
	}
	res, err := m.runner.Run(ctx, "docker", "compose", "--project-directory", dir, "-p", project, "up", "-d")
	out := res.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("docker compose up: %w", err)
	}
	return out, nil
}

// ComposeDown stops and removes a compose project.
func (m *Manager) ComposeDown(ctx context.Context, dir, project string) (string, error) {
	if !validName(project) {
		return "", fmt.Errorf("invalid project name")
	}
	res, err := m.runner.Run(ctx, "docker", "compose", "--project-directory", dir, "-p", project, "down")
	out := res.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("docker compose down: %w", err)
	}
	return out, nil
}

// ComposePs returns the status of a compose project's services.
func (m *Manager) ComposePs(ctx context.Context, dir, project string) (string, error) {
	if !validName(project) {
		return "", fmt.Errorf("invalid project name")
	}
	res, err := m.runner.Run(ctx, "docker", "compose", "--project-directory", dir, "-p", project, "ps")
	return res.CombinedOutput(), err
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

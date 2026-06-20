package api

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jud3ns0hn/digitarlopanel/backend/internal/model"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/pkg/system"
)

// aiTool is a panel capability exposed to the AI assistant and the MCP server.
// Tools are deliberately limited to read and safe-control actions — no deletes,
// no shell, no file writes, no user management — so that granting an assistant
// or external agent access cannot destroy data.
type aiTool struct {
	Name        string
	Description string
	// Schema is the JSON Schema for the tool's input object.
	Schema map[string]any
	// Write marks state-changing tools; they require operator/admin role.
	Write bool
	// Run executes the tool. args is the decoded input object.
	Run func(ctx context.Context, s *Server, args map[string]any) (string, error)
}

// aiTools returns the shared tool registry.
func aiTools() []aiTool {
	objSchema := func(props map[string]any, required ...string) map[string]any {
		if props == nil {
			props = map[string]any{}
		}
		return map[string]any{
			"type":                 "object",
			"properties":           props,
			"required":             required,
			"additionalProperties": false,
		}
	}

	return []aiTool{
		{
			Name:        "get_system_overview",
			Description: "Get host information and current resource usage (CPU, memory, load, disks, uptime).",
			Schema:      objSchema(nil),
			Run: func(ctx context.Context, s *Server, _ map[string]any) (string, error) {
				m, _ := system.Collect(ctx, 300*time.Millisecond)
				return toJSON(map[string]any{
					"os":      s.os,
					"host":    system.Host(ctx),
					"metrics": m,
				})
			},
		},
		{
			Name:        "list_websites",
			Description: "List the Nginx websites managed by the panel (domain, root, PHP version, proxy target, enabled).",
			Schema:      objSchema(nil),
			Run: func(_ context.Context, s *Server, _ map[string]any) (string, error) {
				var sites []model.Website
				s.db.Order("id").Find(&sites)
				return toJSON(sites)
			},
		},
		{
			Name:        "list_databases",
			Description: "List the MySQL/MariaDB databases managed by the panel.",
			Schema:      objSchema(nil),
			Run: func(_ context.Context, s *Server, _ map[string]any) (string, error) {
				var dbs []model.DatabaseInstance
				s.db.Order("id").Find(&dbs)
				return toJSON(dbs)
			},
		},
		{
			Name:        "list_software",
			Description: "List the curated software catalog (Nginx, MariaDB, Redis, PHP-FPM) with install and running status.",
			Schema:      objSchema(nil),
			Run: func(ctx context.Context, s *Server, _ map[string]any) (string, error) {
				out := make([]map[string]any, 0, len(catalog))
				for _, app := range catalog {
					st := s.service.Status(ctx, app.Unit)
					out = append(out, map[string]any{
						"key":       app.Key,
						"name":      app.Name,
						"installed": s.pkg.IsInstalled(ctx, app.pkgFor(s.os.Family)),
						"active":    st.Active,
						"enabled":   st.Enabled,
					})
				}
				return toJSON(out)
			},
		},
		{
			Name:        "service_status",
			Description: "Get the status of a systemd unit by name (e.g. 'nginx', 'mariadb').",
			Schema: objSchema(map[string]any{
				"unit": map[string]any{"type": "string", "description": "systemd unit name"},
			}, "unit"),
			Run: func(ctx context.Context, s *Server, args map[string]any) (string, error) {
				unit, _ := args["unit"].(string)
				if !unitPattern.MatchString(unit) {
					return "", fmt.Errorf("invalid unit name")
				}
				return toJSON(s.service.Status(ctx, unit))
			},
		},
		{
			Name:        "control_service",
			Description: "Start, stop or restart a systemd unit. A safe control action (not a deletion).",
			Write:       true,
			Schema: objSchema(map[string]any{
				"unit":   map[string]any{"type": "string"},
				"action": map[string]any{"type": "string", "enum": []string{"start", "stop", "restart"}},
			}, "unit", "action"),
			Run: func(ctx context.Context, s *Server, args map[string]any) (string, error) {
				unit, _ := args["unit"].(string)
				action, _ := args["action"].(string)
				if !unitPattern.MatchString(unit) {
					return "", fmt.Errorf("invalid unit name")
				}
				var (
					out string
					err error
				)
				switch action {
				case "start":
					out, err = s.service.Start(ctx, unit)
				case "stop":
					out, err = s.service.Stop(ctx, unit)
				case "restart":
					out, err = s.service.Restart(ctx, unit)
				default:
					return "", fmt.Errorf("invalid action")
				}
				if err != nil {
					return "", err
				}
				return "ok: " + out, nil
			},
		},
		{
			Name:        "list_docker_containers",
			Description: "List Docker containers (name, image, state).",
			Schema:      objSchema(nil),
			Run: func(ctx context.Context, s *Server, _ map[string]any) (string, error) {
				if !s.docker.Available() {
					return "docker is not installed", nil
				}
				c, err := s.docker.Containers(ctx)
				if err != nil {
					return "", err
				}
				return toJSON(c)
			},
		},
		{
			Name:        "read_log",
			Description: "Read recent log lines from a systemd unit's journal. Read-only.",
			Schema: objSchema(map[string]any{
				"unit":  map[string]any{"type": "string", "description": "systemd unit, e.g. nginx.service"},
				"lines": map[string]any{"type": "integer", "description": "number of lines (default 100)"},
			}, "unit"),
			Run: func(ctx context.Context, s *Server, args map[string]any) (string, error) {
				unit, _ := args["unit"].(string)
				if !unitPattern.MatchString(unit) {
					return "", fmt.Errorf("invalid unit name")
				}
				lines := 100
				if v, ok := args["lines"].(float64); ok && v > 0 {
					lines = int(v)
				}
				if lines > maxLogLines {
					lines = maxLogLines
				}
				res, err := s.runner.Run(ctx, "journalctl", "-u", unit, "-n", fmt.Sprint(lines), "--no-pager", "--output", "short-iso")
				if err != nil {
					return res.CombinedOutput(), err
				}
				return res.Stdout, nil
			},
		},
		{
			Name:        "list_backups",
			Description: "List backups created by the panel.",
			Schema:      objSchema(nil),
			Run: func(_ context.Context, s *Server, _ map[string]any) (string, error) {
				var b []model.Backup
				s.db.Order("id desc").Limit(50).Find(&b)
				return toJSON(b)
			},
		},
		{
			Name:        "create_backup",
			Description: "Create a backup of a directory (type 'files') or a database (type 'database'). A safe, additive action.",
			Write:       true,
			Schema: objSchema(map[string]any{
				"name":   map[string]any{"type": "string"},
				"type":   map[string]any{"type": "string", "enum": []string{"files", "database"}},
				"source": map[string]any{"type": "string", "description": "directory path or database name"},
			}, "name", "type", "source"),
			Run: func(ctx context.Context, s *Server, args map[string]any) (string, error) {
				name, _ := args["name"].(string)
				typ, _ := args["type"].(string)
				source, _ := args["source"].(string)
				if !backupNamePattern.MatchString(name) {
					return "", fmt.Errorf("invalid backup name")
				}
				rec, err := s.runBackup(ctx, name, typ, source)
				if err != nil {
					return "", err
				}
				return toJSON(rec)
			},
		},
		{
			Name:        "list_firewall_rules",
			Description: "List the allowed firewall ports (ufw/firewalld).",
			Schema:      objSchema(nil),
			Run: func(ctx context.Context, s *Server, _ map[string]any) (string, error) {
				rules, err := s.firewall.List(ctx)
				if err != nil {
					return "no firewall backend available", nil
				}
				return toJSON(rules)
			},
		},
	}
}

// findAITool returns the tool with the given name.
func findAITool(name string) (aiTool, bool) {
	for _, t := range aiTools() {
		if t.Name == name {
			return t, true
		}
	}
	return aiTool{}, false
}

func toJSON(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

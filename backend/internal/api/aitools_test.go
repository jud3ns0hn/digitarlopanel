package api

import (
	"testing"

	"github.com/jud3ns0hn/digitarlopanel/backend/internal/model"
)

func TestToolsForRoleHidesWriteFromViewer(t *testing.T) {
	viewer := toolsForRole(model.RoleViewer)
	for _, tool := range viewer {
		if tool.Write {
			t.Errorf("viewer should not see write tool %q", tool.Name)
		}
	}
	operator := toolsForRole(model.RoleOperator)
	if len(operator) <= len(viewer) {
		t.Errorf("operator (%d) should have more tools than viewer (%d)", len(operator), len(viewer))
	}
}

func TestEveryToolHasObjectSchema(t *testing.T) {
	for _, tool := range aiTools() {
		if tool.Schema["type"] != "object" {
			t.Errorf("tool %q schema type = %v, want object", tool.Name, tool.Schema["type"])
		}
		if tool.Run == nil {
			t.Errorf("tool %q has no Run", tool.Name)
		}
	}
}

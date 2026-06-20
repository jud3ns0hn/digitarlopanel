package pkgmgr

import (
	"context"
	"testing"

	"github.com/jud3ns0hn/digitarlopanel/backend/internal/pkg/osinfo"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/pkg/runner"
)

func TestInstallDispatch(t *testing.T) {
	cases := []struct {
		family   osinfo.Family
		tool     string
		wantCall string
	}{
		{osinfo.FamilyDebian, "apt-get", "apt-get install -y nginx"},
		{osinfo.FamilyRHEL, "dnf", "dnf install -y nginx"},
	}
	for _, tc := range cases {
		t.Run(string(tc.family), func(t *testing.T) {
			mock := &runner.Mock{}
			m := &Manager{family: tc.family, tool: tc.tool, runner: mock}
			if _, err := m.Install(context.Background(), "nginx"); err != nil {
				t.Fatalf("install: %v", err)
			}
			if len(mock.Calls) != 1 || mock.Calls[0] != tc.wantCall {
				t.Fatalf("calls = %v, want [%q]", mock.Calls, tc.wantCall)
			}
		})
	}
}

func TestUnsupportedFamily(t *testing.T) {
	m := &Manager{family: osinfo.FamilyUnknown, runner: &runner.Mock{}}
	if _, err := m.Install(context.Background(), "nginx"); err == nil {
		t.Fatal("expected error for unknown family")
	}
}

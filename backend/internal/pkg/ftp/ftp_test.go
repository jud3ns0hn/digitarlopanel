package ftp

import (
	"context"
	"testing"

	"github.com/jud3ns0hn/digitarlopanel/backend/internal/pkg/runner"
)

func TestCreateBuildsCommand(t *testing.T) {
	mock := &runner.Mock{}
	m := &Manager{runner: mock, sysUser: "ftpuser", sysGroup: "ftpgroup", available: true}
	if _, err := m.Create(context.Background(), "alice", "/srv/ftp/alice", "secret"); err != nil {
		t.Fatal(err)
	}
	want := "pure-pw useradd alice -u ftpuser -g ftpgroup -d /srv/ftp/alice -m"
	if len(mock.Calls) != 1 || mock.Calls[0] != want {
		t.Fatalf("calls = %v, want [%q]", mock.Calls, want)
	}
}

func TestPasswordStdin(t *testing.T) {
	if got := passwordStdin("x"); got != "x\nx\n" {
		t.Errorf("passwordStdin = %q", got)
	}
}

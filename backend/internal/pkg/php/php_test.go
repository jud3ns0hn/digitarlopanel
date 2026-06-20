package php

import (
	"testing"

	"github.com/jud3ns0hn/digitarlopanel/backend/internal/pkg/osinfo"
)

func TestDebianSocketPath(t *testing.T) {
	if got := DebianSocketPath("8.3"); got != "/run/php/php8.3-fpm.sock" {
		t.Errorf("DebianSocketPath = %q", got)
	}
}

func TestPackageName(t *testing.T) {
	if got := PackageName(osinfo.FamilyDebian, "8.3"); got != "php8.3-fpm" {
		t.Errorf("debian package = %q", got)
	}
	if got := PackageName(osinfo.FamilyRHEL, "8.3"); got != "php83-php-fpm" {
		t.Errorf("rhel package = %q", got)
	}
}

func TestListDebianParsesVersions(t *testing.T) {
	dir := t.TempDir()
	// version 8.3 with fpm, 8.1 with fpm, and a bogus "misc" dir.
	for _, v := range []string{"8.3", "8.1"} {
		if err := mkFPM(dir, v); err != nil {
			t.Fatal(err)
		}
	}
	if err := mkDir(dir, "misc"); err != nil {
		t.Fatal(err)
	}
	versions := listDebian(dir)
	if len(versions) != 2 {
		t.Fatalf("expected 2 versions, got %d (%v)", len(versions), versions)
	}
	if versions[0].Version != "8.1" || versions[1].Version != "8.3" {
		t.Errorf("expected sorted [8.1 8.3], got %v", versions)
	}
}

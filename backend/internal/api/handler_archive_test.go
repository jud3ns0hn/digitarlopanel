package api

import (
	"path/filepath"
	"testing"
)

func TestSafeJoin(t *testing.T) {
	dest := filepath.Clean("/srv/extract")
	cases := []struct {
		name string
		ok   bool
	}{
		{"file.txt", true},
		{"sub/dir/file.txt", true},
		{"../escape.txt", false},
		{"../../etc/passwd", false},
		{"sub/../ok.txt", true},
		{"sub/../../escape.txt", false},
	}
	for _, tc := range cases {
		_, ok := safeJoin(dest, tc.name)
		if ok != tc.ok {
			t.Errorf("safeJoin(%q) ok=%v, want %v", tc.name, ok, tc.ok)
		}
	}
}

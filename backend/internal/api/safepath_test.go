package api

import "testing"

func TestResolvePath(t *testing.T) {
	cases := []struct {
		name    string
		root    string
		input   string
		wantErr bool
		want    string
	}{
		{"within root", "/srv/www", "/srv/www/site", false, "/srv/www/site"},
		{"relative within", "/srv/www", "site/index.html", false, "/srv/www/site/index.html"},
		{"traversal escape", "/srv/www", "/srv/www/../../etc/passwd", true, ""},
		{"relative traversal", "/srv/www", "../../etc/passwd", true, ""},
		{"root equals", "/srv/www", "/srv/www", false, "/srv/www"},
		{"unrestricted root", "/", "/etc/passwd", false, "/etc/passwd"},
		{"sibling prefix attack", "/srv/www", "/srv/wwwroot/secret", true, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolvePath(tc.root, tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

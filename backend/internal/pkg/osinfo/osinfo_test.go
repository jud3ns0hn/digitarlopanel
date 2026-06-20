package osinfo

import "testing"

func TestParseFamily(t *testing.T) {
	cases := []struct {
		name    string
		content string
		wantID  string
		want    Family
	}{
		{
			name: "ubuntu",
			content: `NAME="Ubuntu"
VERSION_ID="22.04"
ID=ubuntu
ID_LIKE=debian`,
			wantID: "ubuntu",
			want:   FamilyDebian,
		},
		{
			name: "rocky",
			content: `NAME="Rocky Linux"
VERSION_ID="9.3"
ID="rocky"
ID_LIKE="rhel centos fedora"`,
			wantID: "rocky",
			want:   FamilyRHEL,
		},
		{
			name: "debian",
			content: `PRETTY_NAME="Debian GNU/Linux 12 (bookworm)"
ID=debian`,
			wantID: "debian",
			want:   FamilyDebian,
		},
		{
			name:    "empty",
			content: "",
			wantID:  "",
			want:    FamilyUnknown,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			info := parse(tc.content)
			if info.ID != tc.wantID {
				t.Errorf("ID = %q, want %q", info.ID, tc.wantID)
			}
			if info.Family != tc.want {
				t.Errorf("Family = %q, want %q", info.Family, tc.want)
			}
		})
	}
}

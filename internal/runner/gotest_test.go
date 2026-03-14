package runner

import "testing"

func TestExtractCoverProfile(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"equals single dash", []string{"-coverprofile=cov.out"}, "cov.out"},
		{"space single dash", []string{"-coverprofile", "cov.out"}, "cov.out"},
		{"equals double dash", []string{"--coverprofile=cov.out"}, "cov.out"},
		{"space double dash", []string{"--coverprofile", "cov.out"}, "cov.out"},
		{"not present", []string{"-v", "./..."}, ""},
		{"trailing flag no value", []string{"-coverprofile"}, ""},
		{"mixed args", []string{"-v", "-coverprofile=out.cov", "./..."}, "out.cov"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractCoverProfile(tt.args)
			if got != tt.want {
				t.Errorf("extractCoverProfile(%v) = %q, want %q", tt.args, got, tt.want)
			}
		})
	}
}

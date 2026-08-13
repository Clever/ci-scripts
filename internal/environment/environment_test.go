package environment

import (
	"testing"
)

func TestIsAllowedDeployBranch(t *testing.T) {
	tests := []struct {
		name           string
		circleBranch   string
		deployBranches string
		want           bool
	}{
		{
			name:         "defaults to master, branch is master",
			circleBranch: "master",
			want:         true,
		},
		{
			name:         "defaults to master, branch is main",
			circleBranch: "main",
			want:         false,
		},
		{
			name:           "custom single branch match",
			circleBranch:   "main",
			deployBranches: "main",
			want:           true,
		},
		{
			name:           "custom comma-separated branches match with whitespace",
			circleBranch:   "main",
			deployBranches: "master, main",
			want:           true,
		},
		{
			name:           "custom comma-separated branches no match",
			circleBranch:   "feature/foo",
			deployBranches: "master, main",
			want:           false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			branch = ""
			t.Setenv("CIRCLE_BRANCH", test.circleBranch)
			t.Setenv("DEPLOY_BRANCHES", test.deployBranches)

			if got := IsAllowedDeployBranch(); got != test.want {
				t.Errorf("IsAllowedDeployBranch() = %v, want %v", got, test.want)
			}
		})
	}
}

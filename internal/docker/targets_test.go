package docker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Clever/ci-scripts/internal/repo"
)

// setBuildEnv sets the environment variables BuildTargets reads. The
// environment package caches these on first read, so every test in this
// package must use the same values.
func setBuildEnv(t *testing.T) {
	t.Helper()
	t.Setenv("ECR_ACCOUNT_ID", "123456789012")
	t.Setenv("CIRCLE_SHA1", "abcdef1234567890")
	t.Setenv("CIRCLE_BRANCH", "test-branch")
	t.Setenv("CIRCLE_PROJECT_REPONAME", "my-repo")
}

func TestBuildTargets(t *testing.T) {
	setBuildEnv(t)

	tests := []struct {
		name string
		apps map[string]*repo.AppConfig
		// wantTargetKeys is the set of Dockerfile paths expected as target keys.
		wantTargetKeys []string
		// wantArtifactIDs is the set of app IDs expected in the catapult artifacts.
		wantArtifactIDs []string
		wantErr         bool
	}{
		{
			name: "distinct artifacts with distinct Dockerfile paths each build",
			apps: map[string]*repo.AppConfig{
				"worker-a": {
					Name: "worker-a", RunType: repo.RunTypeDocker,
					ArtifactName: "worker-a", Dockerfile: "Dockerfile.a",
				},
				"worker-b": {
					Name: "worker-b", RunType: repo.RunTypeDocker,
					ArtifactName: "worker-b", Dockerfile: "Dockerfile.b",
				},
			},
			wantTargetKeys:  []string{"Dockerfile.a", "Dockerfile.b"},
			wantArtifactIDs: []string{"worker-a", "worker-b"},
		},
		{
			name: "distinct artifacts sharing the default Dockerfile path errors",
			apps: map[string]*repo.AppConfig{
				"worker-a": {
					Name: "worker-a", RunType: repo.RunTypeDocker,
					ArtifactName: "worker-a",
				},
				"worker-b": {
					Name: "worker-b", RunType: repo.RunTypeDocker,
					ArtifactName: "worker-b",
				},
			},
			wantErr: true,
		},
		{
			name: "distinct artifacts sharing an explicit Dockerfile path errors",
			apps: map[string]*repo.AppConfig{
				"worker-a": {
					Name: "worker-a", RunType: repo.RunTypeDocker,
					ArtifactName: "worker-a", Dockerfile: "Dockerfile.worker",
				},
				"worker-b": {
					Name: "worker-b", RunType: repo.RunTypeDocker,
					ArtifactName: "worker-b", Dockerfile: "Dockerfile.worker",
				},
			},
			wantErr: true,
		},
		{
			name: "apps sharing an artifact name build once",
			apps: map[string]*repo.AppConfig{
				"sso-my-service": {
					Name: "sso-my-service", RunType: repo.RunTypeDocker,
					ArtifactName: "my-service", Dockerfile: "Dockerfile.sso",
				},
				"my-service": {
					Name: "my-service", RunType: repo.RunTypeDocker,
					ArtifactName: "my-service", Dockerfile: "Dockerfile.sso",
				},
			},
			wantTargetKeys:  []string{"Dockerfile.sso"},
			wantArtifactIDs: []string{"sso-my-service", "my-service"},
		},
		{
			name: "lambda apps produce no docker targets",
			apps: map[string]*repo.AppConfig{
				"my-lambda": {
					Name: "my-lambda", RunType: repo.RunTypeLambda, ArtifactName: "my-lambda",
				},
			},
			wantTargetKeys:  []string{},
			wantArtifactIDs: []string{},
		},
		{
			name: "single docker app builds one target",
			apps: map[string]*repo.AppConfig{
				"my-service": {
					Name: "my-service", RunType: repo.RunTypeDocker, ArtifactName: "my-service",
				},
			},
			wantTargetKeys:  []string{""},
			wantArtifactIDs: []string{"my-service"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targets, artifacts, err := BuildTargets(tt.apps)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "shared Dockerfile path")
				assert.Nil(t, targets)
				assert.Nil(t, artifacts)
				return
			}
			require.NoError(t, err)

			gotKeys := make([]string, 0, len(targets))
			for k := range targets {
				gotKeys = append(gotKeys, k)
			}
			assert.ElementsMatch(t, tt.wantTargetKeys, gotKeys)

			gotIDs := make([]string, 0, len(artifacts))
			for _, a := range artifacts {
				gotIDs = append(gotIDs, a.ID)
			}
			assert.ElementsMatch(t, tt.wantArtifactIDs, gotIDs)
		})
	}
}

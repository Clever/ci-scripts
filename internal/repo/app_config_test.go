package repo

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Clever/catapult/gen-go/models"
	"github.com/stretchr/testify/assert"
)

func TestAppConfigForCatapult(t *testing.T) {
	tests := []struct {
		name     string
		appName  string
		lc       *models.LaunchConfig
		expected *AppConfig
	}{
		{
			name:    "no build or run block defaults to docker with app name as artifact",
			appName: "my-service",
			lc:      &models.LaunchConfig{},
			expected: &AppConfig{
				Name:         "my-service",
				RunType:      RunTypeDocker,
				ArtifactName: "my-service",
			},
		},
		{
			name:    "explicit docker run type",
			appName: "my-service",
			lc: &models.LaunchConfig{
				Run: &models.LaunchRun{Type: models.RunTypeDocker},
			},
			expected: &AppConfig{
				Name:         "my-service",
				RunType:      RunTypeDocker,
				ArtifactName: "my-service",
			},
		},
		{
			name:    "lambda run type",
			appName: "my-lambda",
			lc: &models.LaunchConfig{
				Run: &models.LaunchRun{Type: models.RunTypeLambda},
			},
			expected: &AppConfig{
				Name:         "my-lambda",
				RunType:      RunTypeLambda,
				ArtifactName: "my-lambda",
			},
		},
		{
			name:    "artifact name override",
			appName: "sso-my-service",
			lc: &models.LaunchConfig{
				Build: &models.LaunchBuild{
					Artifact: &models.BuildArtifact{Name: "my-service"},
				},
			},
			expected: &AppConfig{
				Name:         "sso-my-service",
				RunType:      RunTypeDocker,
				ArtifactName: "my-service",
			},
		},
		{
			name:    "build command",
			appName: "my-service",
			lc: &models.LaunchConfig{
				Build: &models.LaunchBuild{
					Artifact: &models.BuildArtifact{Command: "make build"},
				},
			},
			expected: &AppConfig{
				Name:         "my-service",
				RunType:      RunTypeDocker,
				ArtifactName: "my-service",
				BuildCommand: "make build",
			},
		},
		{
			name:    "dockerfile override",
			appName: "my-service",
			lc: &models.LaunchConfig{
				Build: &models.LaunchBuild{
					Docker: &models.BuildDocker{File: "Dockerfile.api"},
				},
			},
			expected: &AppConfig{
				Name:         "my-service",
				RunType:      RunTypeDocker,
				ArtifactName: "my-service",
				Dockerfile:   "Dockerfile.api",
			},
		},
		{
			name:    "dependencies",
			appName: "my-service",
			lc: &models.LaunchConfig{
				Build: &models.LaunchBuild{
					Artifact: &models.BuildArtifact{
						Dependencies: []string{"*.go", "go.mod", "go.sum"},
					},
				},
			},
			expected: &AppConfig{
				Name:         "my-service",
				RunType:      RunTypeDocker,
				ArtifactName: "my-service",
				Dependencies: []string{"*.go", "go.mod", "go.sum"},
			},
		},
		{
			name:    "all fields set",
			appName: "sso-my-service",
			lc: &models.LaunchConfig{
				Run: &models.LaunchRun{Type: models.RunTypeDocker},
				Build: &models.LaunchBuild{
					Artifact: &models.BuildArtifact{
						Name:         "my-service",
						Command:      "make build",
						Dependencies: []string{"*.go", "Makefile"},
					},
					Docker: &models.BuildDocker{File: "Dockerfile.sso"},
				},
			},
			expected: &AppConfig{
				Name:         "sso-my-service",
				RunType:      RunTypeDocker,
				ArtifactName: "my-service",
				BuildCommand: "make build",
				Dockerfile:   "Dockerfile.sso",
				Dependencies: []string{"*.go", "Makefile"},
			},
		},
		{
			name:    "build block present but artifact is nil",
			appName: "my-service",
			lc: &models.LaunchConfig{
				Build: &models.LaunchBuild{
					Docker: &models.BuildDocker{File: "Dockerfile.two"},
				},
			},
			expected: &AppConfig{
				Name:         "my-service",
				RunType:      RunTypeDocker,
				ArtifactName: "my-service",
				Dockerfile:   "Dockerfile.two",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := appConfigForCatapult(tt.appName, tt.lc)
			assert.Equal(t, tt.expected, got)
		})
	}
}

// writeStackYAML creates config/<appName>/stack.yaml under dir with the given content.
func writeStackYAML(t *testing.T, dir, appName, content string) {
	t.Helper()
	appDir := filepath.Join(dir, "config", appName)
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("mkdir %s: %v", appDir, err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "stack.yaml"), []byte(content), 0644); err != nil {
		t.Fatalf("write stack.yaml: %v", err)
	}
}

func TestAppConfigForKubernetes(t *testing.T) {
	tests := []struct {
		name        string
		appName     string
		stackYAML   string
		expected    *AppConfig
		expectError bool
	}{
		{
			name:    "no build block defaults to docker with app name as artifact",
			appName: "my-service",
			stackYAML: `
stack:
  helm:
    chart: clever-application
`,
			expected: &AppConfig{
				Name:         "my-service",
				RunType:      RunTypeDocker,
				ArtifactName: "my-service",
			},
		},
		{
			name:    "clever-lambda chart sets lambda run type",
			appName: "my-lambda",
			stackYAML: `
stack:
  helm:
    chart: clever-lambda
`,
			expected: &AppConfig{
				Name:         "my-lambda",
				RunType:      RunTypeLambda,
				ArtifactName: "my-lambda",
			},
		},
		{
			name:    "artifact name override",
			appName: "sso-my-service",
			stackYAML: `
stack:
  helm:
    chart: clever-application
build:
  artifactName: my-service
`,
			expected: &AppConfig{
				Name:         "sso-my-service",
				RunType:      RunTypeDocker,
				ArtifactName: "my-service",
			},
		},
		{
			name:    "build command",
			appName: "my-service",
			stackYAML: `
stack:
  helm:
    chart: clever-application
build:
  command: make build
`,
			expected: &AppConfig{
				Name:         "my-service",
				RunType:      RunTypeDocker,
				ArtifactName: "my-service",
				BuildCommand: "make build",
			},
		},
		{
			name:    "dockerfile override",
			appName: "my-service",
			stackYAML: `
stack:
  helm:
    chart: clever-application
build:
  dockerfile: Dockerfile.api
`,
			expected: &AppConfig{
				Name:         "my-service",
				RunType:      RunTypeDocker,
				ArtifactName: "my-service",
				Dockerfile:   "Dockerfile.api",
			},
		},
		{
			name:    "dependencies",
			appName: "my-service",
			stackYAML: `
stack:
  helm:
    chart: clever-application
build:
  dependencies:
    - "*.go"
    - go.mod
    - go.sum
`,
			expected: &AppConfig{
				Name:         "my-service",
				RunType:      RunTypeDocker,
				ArtifactName: "my-service",
				Dependencies: []string{"*.go", "go.mod", "go.sum"},
			},
		},
		{
			name:    "all fields set",
			appName: "sso-my-service",
			stackYAML: `
stack:
  helm:
    chart: clever-application
build:
  artifactName: my-service
  command: make build
  dockerfile: Dockerfile.sso
  dependencies:
    - "*.go"
    - Makefile
`,
			expected: &AppConfig{
				Name:         "sso-my-service",
				RunType:      RunTypeDocker,
				ArtifactName: "my-service",
				BuildCommand: "make build",
				Dockerfile:   "Dockerfile.sso",
				Dependencies: []string{"*.go", "Makefile"},
			},
		},
		{
			name:        "missing stack.yaml returns error",
			appName:     "nonexistent-app",
			stackYAML:   "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)

			if tt.stackYAML != "" {
				writeStackYAML(t, dir, tt.appName, tt.stackYAML)
			}

			got, err := appConfigForKubernetes(tt.appName)
			if tt.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestDiscoverKubernetesApplications(t *testing.T) {
	t.Run("discovers all apps from config directory", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)

		writeStackYAML(t, dir, "app-one", `
stack:
  helm:
    chart: clever-application
build:
  artifactName: app-one
`)
		writeStackYAML(t, dir, "app-two", `
stack:
  helm:
    chart: clever-lambda
build:
  command: make build
`)

		apps, err := discoverKubernetesApplications("config")
		assert.NoError(t, err)
		assert.Len(t, apps, 2)
		assert.Equal(t, &AppConfig{
			Name:         "app-one",
			RunType:      RunTypeDocker,
			ArtifactName: "app-one",
		}, apps["app-one"])
		assert.Equal(t, &AppConfig{
			Name:         "app-two",
			RunType:      RunTypeLambda,
			ArtifactName: "app-two",
			BuildCommand: "make build",
		}, apps["app-two"])
	})

	t.Run("non-directory entries are skipped", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)

		writeStackYAML(t, dir, "my-app", `
stack:
  helm:
    chart: clever-application
`)
		// Write a non-directory file directly under config/
		configDir := filepath.Join(dir, "config")
		if err := os.WriteFile(filepath.Join(configDir, "not-an-app.yaml"), []byte("foo: bar"), 0644); err != nil {
			t.Fatalf("write stray file: %v", err)
		}

		apps, err := discoverKubernetesApplications("config")
		assert.NoError(t, err)
		assert.Len(t, apps, 1)
		assert.Contains(t, apps, "my-app")
	})

	t.Run("empty config directory returns empty map", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)

		if err := os.MkdirAll(filepath.Join(dir, "config"), 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}

		apps, err := discoverKubernetesApplications("config")
		assert.NoError(t, err)
		assert.Empty(t, apps)
	})

	t.Run("missing config directory returns error", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)

		_, err := discoverKubernetesApplications("config")
		assert.Error(t, err)
	})
}

package repo

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var fixtureFiles = map[string]string{
	"go.mod":                       "module example.com/fixture\n\ngo 1.21\n",
	"go.sum":                       "",
	"Dockerfile":                   "FROM scratch\n",
	"README.md":                    "# fixture\n",
	".github/workflows/ci.yml":     "on: push\n",
	"shared/shared.go":             "package shared\n\nfunc Name() string { return \"shared\" }\n",
	"other/other.go":               "package other\n\nfunc Name() string { return \"other\" }\n",
	"cmd/app-one/static/index.txt": "hello\n",
	"cmd/app-one/main.go": `package main

import (
	_ "embed"

	"example.com/fixture/shared"
)

//go:embed static/index.txt
var index string

func main() { println(shared.Name(), index) }
`,
	"cmd/app-two/main.go": `package main

import "example.com/fixture/other"

func main() { println(other.Name()) }
`,
	"config/app-one/stack.yaml":   "build:\n  package: ./cmd/app-one\n",
	"config/app-one/values.yaml":  "replicas: 1\n",
	"config/app-two/stack.yaml":   "build:\n  package: cmd/app-two\n",
	"config/app-two/values.yaml":  "replicas: 1\n",
	"config/app-three/stack.yaml": "build:\n  command: make build\n",
}

func TestDiscoverKubernetesApplicationsByImportGraph(t *testing.T) {
	tests := []struct {
		name         string
		changedFiles map[string]string
		selection    ChangeSelection
		expectedApps []string
	}{
		{
			name:         "shared package change selects only the app that imports it",
			changedFiles: map[string]string{"shared/shared.go": "package shared\n\nfunc Name() string { return \"changed\" }\n"},
			selection:    SelectByImportGraph,
			expectedApps: []string{"app-one", "app-three"},
		},
		{
			name:         "new file in an imported package selects the app",
			changedFiles: map[string]string{"other/extra.go": "package other\n"},
			selection:    SelectByImportGraph,
			expectedApps: []string{"app-three", "app-two"},
		},
		{
			name:         "test file in an imported package selects no import-graph app",
			changedFiles: map[string]string{"shared/shared_test.go": "package shared\n"},
			selection:    SelectByImportGraph,
			expectedApps: []string{"app-three"},
		},
		{
			name:         "embedded file change selects the app",
			changedFiles: map[string]string{"cmd/app-one/static/index.txt": "changed\n"},
			selection:    SelectByImportGraph,
			expectedApps: []string{"app-one", "app-three"},
		},
		{
			name:         "values.yaml change selects only that app",
			changedFiles: map[string]string{"config/app-two/values.yaml": "replicas: 2\n"},
			selection:    SelectByImportGraph,
			expectedApps: []string{"app-three", "app-two"},
		},
		{
			name:         "go.mod change selects every import-graph app",
			changedFiles: map[string]string{"go.mod": "module example.com/fixture\n\ngo 1.22\n"},
			selection:    SelectByImportGraph,
			expectedApps: []string{"app-one", "app-three", "app-two"},
		},
		{
			name:         "Dockerfile change selects docker apps",
			changedFiles: map[string]string{"Dockerfile": "FROM alpine\n"},
			selection:    SelectByImportGraph,
			expectedApps: []string{"app-one", "app-three", "app-two"},
		},
		{
			name: "README and .github changes select no import-graph app",
			changedFiles: map[string]string{
				"README.md":                "# changed\n",
				".github/workflows/ci.yml": "on: pull_request\n",
			},
			selection:    SelectByImportGraph,
			expectedApps: []string{"app-three"},
		},
		{
			name:         "dependency selection ignores build.package",
			changedFiles: map[string]string{"README.md": "# changed\n"},
			selection:    SelectByDependencies,
			expectedApps: []string{"app-one", "app-three", "app-two"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setupFixtureRepo(t, test.changedFiles)

			apps, err := discoverKubernetesApplications("config", test.selection)
			require.NoError(t, err)
			assert.Equal(t, test.expectedApps, sortedAppNames(apps))
		})
	}
}

func TestChangePathFinderFailsForMissingPackage(t *testing.T) {
	setupFixtureRepo(t, map[string]string{"README.md": "# changed\n"})

	_, err := newChangePathFinder(SelectByImportGraph).paths(&AppConfig{Name: "app-one", Package: "./cmd/missing"})
	assert.Error(t, err)
}

func TestChangePathFinderSkipsMissingDependency(t *testing.T) {
	setupFixtureRepo(t, map[string]string{
		"cmd/app-two/main.go": `package main

import (
	"example.com/fixture/gen"
	"example.com/fixture/other"
)

func main() { println(other.Name(), gen.Name()) }
`,
	})

	apps, err := discoverKubernetesApplications("config", SelectByImportGraph)
	require.NoError(t, err)
	assert.Equal(t, []string{"app-three", "app-two"}, sortedAppNames(apps))
}

func TestChangePathFinderListsSharedPackageOnce(t *testing.T) {
	setupFixtureRepo(t, map[string]string{"README.md": "# changed\n"})
	finder := newChangePathFinder(SelectByImportGraph)

	east, err := finder.paths(&AppConfig{Name: "app-east", RunType: RunTypeLambda, Package: "./cmd/app-one"})
	require.NoError(t, err)
	west, err := finder.paths(&AppConfig{Name: "app-west", RunType: RunTypeLambda, Package: "cmd/app-one"})
	require.NoError(t, err)

	assert.Len(t, finder.goInputsByPackage, 1)
	assert.Contains(t, east, "config/app-east")
	assert.Contains(t, west, "config/app-west")
	assert.NotContains(t, west, "config/app-east")
}

// setupFixtureRepo commits fixtureFiles, then commits changedFiles, and compares the two commits.
func setupFixtureRepo(t *testing.T, changedFiles map[string]string) {
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("CIRCLE_BRANCH", "feature")
	t.Setenv("MASTER_COMPARE", "HEAD~1..HEAD")

	writeFiles(t, dir, fixtureFiles)
	runGit(t, "init", "-q")
	runGit(t, "add", "-A")
	runGit(t, "commit", "-q", "-m", "fixture")
	writeFiles(t, dir, changedFiles)
	runGit(t, "add", "-A")
	runGit(t, "commit", "-q", "-m", "change")
}

func writeFiles(t *testing.T, dir string, files map[string]string) {
	for name, content := range files {
		filePath := filepath.Join(dir, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(filePath), 0755))
		require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))
	}
}

func runGit(t *testing.T, args ...string) {
	command := exec.Command("git", append([]string{"-c", "user.name=test", "-c", "user.email=test@example.com"}, args...)...)
	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))
}

func sortedAppNames(apps map[string]*AppConfig) []string {
	names := []string{}
	for name := range apps {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

package repo

import (
	"os/exec"
	"strings"
	"testing"
)

func TestDetectArtifactDependencyChange(t *testing.T) {
	// Uncomment this if you want to debug this function. Note that the
	// shell commands are running within this directory which limits
	// find from seeing all files in this repo.
	t.Skip("skipping test")

	changed, err := DetectArtifactDependencyChange([]string{"*.go"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Fatal("expected changed to be true")
	}
}

func TestDependenciesChangedInRange(t *testing.T) {
	tests := []struct {
		name         string
		localMaster  bool
		compareRange string
		changedFiles []string
		dependencies []string
		wantChanged  bool
	}{
		{
			name:         "dependency changed, no local master",
			compareRange: "master...feature",
			changedFiles: []string{"deps/lib.go"},
			dependencies: []string{"deps/*"},
			wantChanged:  true,
		},
		{
			name:         "dependency untouched, no local master",
			compareRange: "master...feature",
			changedFiles: []string{"other/main.go"},
			dependencies: []string{"deps/*"},
			wantChanged:  false,
		},
		{
			name:         "dependency changed, local master",
			localMaster:  true,
			compareRange: "master...feature",
			changedFiles: []string{"deps/lib.go"},
			dependencies: []string{"deps/*"},
			wantChanged:  true,
		},
		{
			name:         "dependency untouched, local master",
			localMaster:  true,
			compareRange: "master...feature",
			changedFiles: []string{"other/main.go"},
			dependencies: []string{"deps/*"},
			wantChanged:  false,
		},
		{
			name:         "sha range is unchanged",
			compareRange: "HEAD^1..HEAD",
			changedFiles: []string{"deps/lib.go"},
			dependencies: []string{"deps/*"},
			wantChanged:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			newCircleCIStyleCheckout(t, test.localMaster, test.changedFiles)

			changed, err := dependenciesChangedInRange(test.compareRange, test.dependencies)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if changed != test.wantChanged {
				t.Fatalf("changed = %v, want %v", changed, test.wantChanged)
			}
		})
	}
}

func TestDependenciesChangedInRangeIncludesGitStderr(t *testing.T) {
	newCircleCIStyleCheckout(t, false, []string{"deps/lib.go"})

	_, err := dependenciesChangedInRange("missing...feature", []string{"deps/*"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "bad revision") {
		t.Fatalf("error %q does not include git stderr", err)
	}
}

func TestResolveCompareRange(t *testing.T) {
	newCircleCIStyleCheckout(t, false, []string{"deps/lib.go"})

	tests := map[string]string{
		"master...feature": "origin/master...feature",
		"master..feature":  "origin/master..feature",
		"HEAD^1..HEAD":     "HEAD^1..HEAD",
		"missing...x":      "missing...x",
		"no-range":         "no-range",
	}
	for compareRange, want := range tests {
		if got := resolveCompareRange(compareRange); got != want {
			t.Errorf("resolveCompareRange(%q) = %q, want %q", compareRange, got, want)
		}
	}
}

// newCircleCIStyleCheckout chdirs into a clone that has origin/master and a
// checked-out feature branch. A local master exists only when localMaster is set.
func newCircleCIStyleCheckout(t *testing.T, localMaster bool, featureChangedFiles []string) {
	t.Helper()
	t.Setenv("GIT_AUTHOR_NAME", "test")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "test")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")

	upstream := t.TempDir()
	runGit(t, upstream, "init", "-q", "-b", "master")
	runGit(t, upstream, "commit", "-q", "--allow-empty", "-m", "initial")
	runGit(t, upstream, "checkout", "-q", "-b", "feature")
	for _, file := range featureChangedFiles {
		runGit(t, upstream, "commit", "-q", "--allow-empty", "-m", "touch "+file)
		writeAndAdd(t, upstream, file)
	}

	checkout := t.TempDir()
	runGit(t, checkout, "clone", "-q", "--no-checkout", upstream, ".")
	runGit(t, checkout, "checkout", "-q", "-B", "feature", "origin/feature")
	if localMaster {
		runGit(t, checkout, "branch", "master", "origin/master")
	}
	t.Chdir(checkout)
}

func writeAndAdd(t *testing.T, directory, file string) {
	t.Helper()
	runGit(t, directory, "update-index", "--add", "--cacheinfo", "100644,"+hashBlob(t, directory, file)+","+file)
	runGit(t, directory, "commit", "-q", "-m", "add "+file)
}

func hashBlob(t *testing.T, directory, content string) string {
	t.Helper()
	command := exec.Command("git", "hash-object", "-w", "--stdin")
	command.Dir = directory
	command.Stdin = strings.NewReader(content)
	output, err := command.Output()
	if err != nil {
		t.Fatalf("git hash-object: %v", err)
	}
	return strings.TrimSpace(string(output))
}

func runGit(t *testing.T, directory string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = directory
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
}

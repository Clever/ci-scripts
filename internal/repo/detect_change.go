package repo

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/Clever/ci-scripts/internal/environment"
)

const remoteName = "origin"

// DetectArtifactDependencyChange checks if any of the given file globs have
// changed since the primary branch. An empty slice means no filtering —
// the app is always considered changed.
func DetectArtifactDependencyChange(dependencies []string) (bool, error) {
	if len(dependencies) == 0 {
		return true, nil
	}

	compareRange := environment.PrimaryCompare()
	if environment.Branch() == "master" {
		compareRange = environment.PreviousPipelineCompare()
	}

	return dependenciesChangedInRange(compareRange, dependencies)
}

func dependenciesChangedInRange(compareRange string, dependencies []string) (bool, error) {
	args := append([]string{"diff", "--name-only", resolveCompareRange(compareRange), "--"}, dependencies...)
	gitCmd := exec.Command("git", args...)
	fmt.Println("Checking for changes with:", gitCmd.String())

	output, err := gitCmd.Output()
	if err != nil {
		return false, fmt.Errorf("git diff: %v: %s", err, stderrOf(err))
	}
	fmt.Println("Changed files:", string(output))

	return len(output) != 0, nil
}

// CircleCI checkouts have remote-tracking refs but no local branch for the
// primary branch, so a range like master...feature needs origin/master.
func resolveCompareRange(compareRange string) string {
	separator := "..."
	base, head, found := strings.Cut(compareRange, separator)
	if !found {
		separator = ".."
		base, head, found = strings.Cut(compareRange, separator)
	}
	if !found || revisionExists(base) {
		return compareRange
	}

	remoteBase := remoteName + "/" + base
	if !revisionExists(remoteBase) {
		return compareRange
	}
	return remoteBase + separator + head
}

func revisionExists(revision string) bool {
	return exec.Command("git", "rev-parse", "--verify", "--quiet", revision+"^{commit}").Run() == nil
}

func stderrOf(err error) string {
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		return strings.TrimSpace(string(exitError.Stderr))
	}
	return ""
}

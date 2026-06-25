package repo

import (
	"fmt"
	"os/exec"

	"github.com/Clever/ci-scripts/internal/environment"
)

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

	args := append([]string{"diff", "--name-only", compareRange, "--"}, dependencies...)
	gitCmd := exec.Command("git", args...)
	fmt.Println("Checking for changes with:", gitCmd.String())

	output, err := gitCmd.Output()
	if err != nil {
		return false, fmt.Errorf("git diff: %v", err)
	}
	fmt.Println("Changed files:", string(output))

	return len(output) != 0, nil
}

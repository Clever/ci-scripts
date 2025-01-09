package repo

import (
	"fmt"
	"os/exec"

	"github.com/Clever/catapult/gen-go/models"
)

// DetectArtifactDependencyChange checks if the artifact dependency
// globs defined in the launch config have changed by using git diff for
// only the specified file globs. The dependencies are always checked
// against the primary branch. More advanced dependency checking is
// hard and involves persisted caching of some sort which should be
// left to a build system later on.
func DetectArtifactDependencyChange(lc *models.LaunchConfig) (bool, error) {
	if lc.Build == nil || lc.Build.Artifact == nil || lc.Build.Artifact.Dependencies == nil {
		return true, nil
	}

	args := append([]string{"diff", "--name-only", "HEAD", "master", "--"}, lc.Build.Artifact.Dependencies...)
	gitCmd := exec.Command("git", args...)
	fmt.Println(gitCmd.String())

	output, err := gitCmd.Output()
	if err != nil {
		return false, fmt.Errorf("git diff: %v", err)
	}
	fmt.Println(string(output))

	return string(output) != "", nil
}

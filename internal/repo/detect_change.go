package repo

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/Clever/catapult/gen-go/models"
)

// go list -f '{{.Dir}}' -deps ./cmd/arkdb | grep $(pwd) | grep -v '/vendor/'

// DetectArtifactDependencyChange checks if the artifact dependency
// globs defined in the launch config have changed by using git diff for
// only the specified file globs. The dependencies are always checked
// against the base branch (HEAD). More advanced dependency checking is
// hard and involves persisted caching of some sort which should be
// left to a build system later on.
func DetectArtifactDependencyChange(lc *models.LaunchConfig) (bool, error) {
	if lc.Build == nil || lc.Build.Artifact == nil || lc.Build.Artifact.Dependencies == nil {
		return true, nil
	}

	// Define the find command to search for multiple globs
	globs := []string{".", "-type", "f", "("}
	nots := []string{}
	for _, glob := range lc.Build.Artifact.Dependencies {
		if strings.HasPrefix(glob, "!") {
			nots = append(nots, "-not", "-path", glob[1:])
			continue
		}

		if len(globs) == 4 {
			globs = append(globs, "-path", glob)
		} else {
			globs = append(globs, "-o", "-path", glob)
		}
	}
	globs = append(globs, ")")
	globs = append(globs, nots...)
	findCmd := exec.Command("find", globs...)
	fmt.Println(findCmd.String())

	// Capture the output from find
	output, err := findCmd.Output()
	if err != nil {
		return false, fmt.Errorf("find: %v", err)
	}

	// Split the output into file paths
	files := strings.Fields(string(output))
	if len(files) == 0 {
		return false, errors.New("no matching files")
	}
	// You can uncomment the following line to see the files found by
	// the find command but it is very verbose
	fmt.Println(files)

	// Prepare git diff command with the found files
	args := append([]string{"diff", "--name-only", "HEAD", "master", "--"}, files...)
	gitCmd := exec.Command("git", args...)
	// This command is also very verbose
	// fmt.Println(gitCmd.String())

	// Capture the output from git diff
	output, err = gitCmd.Output()
	if err != nil {
		return false, fmt.Errorf("git diff: %v", err)
	}
	fmt.Println(string(output))

	return string(output) != "", nil
}

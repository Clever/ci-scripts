package docker

import (
	"fmt"

	"github.com/Clever/catapult/gen-go/models"
	"github.com/Clever/ci-scripts/internal/environment"
	"github.com/Clever/ci-scripts/internal/repo"
)

// BuildTargets returns a map of dockerfile path keys with their
// associated tags for pushing to a remote repository. If multiple apps
// share a repository then only the first matching Dockerfile and its
// set of tags will be in the final list. This is an optimization so we
// do not build multiple copies of the same Dockerfile which only differ
// at runtime.
func BuildTargets(apps map[string]*models.LaunchConfig) map[string][]string {
	targets := map[string][]string{}
	done := map[string]struct{}{}

	for name, launch := range apps {
		if !repo.IsDockerRunType(launch) {
			continue
		}

		artifact := repo.ArtifactName(name, launch)
		// Any apps with a shared artifact only need to be built and
		// tagged once.
		if _, ok := done[artifact]; ok {
			fmt.Println("shared artifact", artifact)
			continue
		}
		done[artifact] = struct{}{}

		tags := []string{}
		for _, region := range environment.AWSRegions {
			tag := fmt.Sprintf(
				"%s.dkr.ecr.%s.amazonaws.com/%s:%s",
				environment.ECRAccountID, region, artifact, environment.ShortSHA1,
			)
			tags = append(tags, tag)
		}
		targets[repo.Dockerfile(launch)] = tags
	}
	return targets
}

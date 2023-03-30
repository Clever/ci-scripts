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
loop:
	for name, launch := range apps {
		if !repo.IsDockerRunType(launch) {
			continue
		}

		tags := []string{}
		for _, region := range environment.AWSRegions {
			tag := fmt.Sprintf(
				"%s.dkr.ecr.%s.amazonaws.com/%s:%s",
				environment.ECRAccountID, region, repo.ArtifactName(name, launch), environment.ShortSHA1,
			)

			// Any apps with a shared artifact will have the same
			// set of tags. If we find just one of the tags in this
			// map that means we have already built and pushed this
			// app and can skip it.
			if _, ok := done[tag]; ok {
				fmt.Println("shared tag", tag)
				continue loop
			}
			tags = append(tags, tag)
			done[tag] = struct{}{}
		}
		targets[repo.Dockerfile(launch)] = tags
	}
	return targets
}

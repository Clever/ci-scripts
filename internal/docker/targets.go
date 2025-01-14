package docker

import (
	"fmt"

	"github.com/Clever/catapult/gen-go/models"
	"github.com/Clever/ci-scripts/internal/catapult"
	"github.com/Clever/ci-scripts/internal/environment"
	"github.com/Clever/ci-scripts/internal/repo"
)

type DockerTarget struct {
	// Tags are the list of tags to push for the built docker image.
	Tags []string
	// Command is the command to run to build the lambda artifact.
	Command string
}

// BuildTargets returns a map of dockerfile path keys with their
// associated tags for pushing to a remote repository. If multiple apps
// share a repository then only the first matching Dockerfile and its
// set of tags will be in the final list. This is an optimization so we
// do not build multiple copies of the same Dockerfile which only differ
// at runtime.
func BuildTargets(apps map[string]*models.LaunchConfig) (map[string]DockerTarget, []*catapult.Artifact) {
	var (
		targets   = map[string]DockerTarget{}
		done      = map[string]struct{}{}
		artifacts []*catapult.Artifact
	)

	for name, launch := range apps {
		if !repo.IsDockerRunType(launch) {
			continue
		}

		artifact := repo.ArtifactName(name, launch)
		artifacts = append(artifacts, &catapult.Artifact{
			RunType:   string(models.RunTypeDocker),
			ID:        name,
			Branch:    environment.Branch,
			Source:    fmt.Sprintf("github:Clever/%s@%s", environment.Repo, environment.FullSHA1),
			Artifacts: fmt.Sprintf("docker:clever/%s@%s", artifact, environment.ShortSHA1),
		})

		// Any apps with a shared artifact only need to be built and
		// tagged once. Short-circuit after we assemble our catapult
		// artifacts because catapult still needs an artifact reference
		// for every app.
		if _, ok := done[artifact]; ok {
			fmt.Println(name, "shares artifact with", artifact)
			continue
		}
		done[artifact] = struct{}{}

		tags := []string{}
		for _, region := range environment.Regions {
			tag := fmt.Sprintf(
				"%s.dkr.ecr.%s.amazonaws.com/%s:%s",
				environment.ECRAccountID, region, artifact, environment.ShortSHA1,
			)
			tags = append(tags, tag)
		}

		targets[repo.Dockerfile(launch)] = DockerTarget{
			Tags:    tags,
			Command: repo.BuildCommand(launch),
		}
	}
	return targets, artifacts
}

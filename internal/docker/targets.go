package docker

import (
	"fmt"

	"github.com/Clever/ci-scripts/internal/catapult"
	"github.com/Clever/ci-scripts/internal/environment"
	"github.com/Clever/ci-scripts/internal/repo"
)

// DockerTarget contains information about how to build and push a
// docker build target.
type DockerTarget struct {
	// Tags are the list of tags to push for the built docker image.
	Tags []string
	// Command is the command to run to build the lambda artifact.
	Command string
}

// BuildTargets returns a map of dockerfile path keys with their build
// command and associated tags for pushing to a remote repository. If
// multiple apps share an artifact name then only the first matching
// Dockerfile and its set of tags will be in the final list. This is an
// optimization so we do not build multiple copies of the same image
// which only differ at runtime.
//
// It returns an error when two apps publish to different artifacts but
// resolve to the same Dockerfile path. The targets map is keyed by
// Dockerfile path, so the second app would silently overwrite the
// first and one image would never be built or pushed. Because each app
// compiles and copies in its own binary, a shared Dockerfile path
// cannot produce the distinct images those apps need, so we fail loudly
// instead. Give each app its own build.docker.file to resolve it.
func BuildTargets(apps map[string]*repo.AppConfig) (map[string]DockerTarget, []*catapult.Artifact, error) {
	// dockerfileOwner is the first app/artifact to claim a Dockerfile path.
	type dockerfileOwner struct{ app, artifact string }
	var (
		targets   = map[string]DockerTarget{}
		done      = map[string]struct{}{}
		owner     = map[string]dockerfileOwner{}
		artifacts []*catapult.Artifact
	)

	for name, launch := range apps {
		if !repo.IsDockerRunType(launch) {
			continue
		}

		artifact := launch.ArtifactName
		artifacts = append(artifacts, &catapult.Artifact{
			RunType:   launch.RunType,
			ID:        name,
			Branch:    environment.Branch(),
			Source:    fmt.Sprintf("github:Clever/%s@%s", environment.Repo(), environment.FullSHA1()),
			Artifacts: fmt.Sprintf("docker:clever/%s@%s", artifact, environment.ShortSHA1()),
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

		// Two apps publishing to different artifacts cannot share a
		// Dockerfile path: keyed by path, the second would overwrite the
		// first and one image would never ship. Fail loudly.
		if prev, ok := owner[launch.Dockerfile]; ok {
			return nil, nil, fmt.Errorf(
				"apps %q and %q both build Dockerfile %q but publish to different "+
					"artifacts (%q and %q); goci cannot build distinct images from a "+
					"shared Dockerfile path. Give each app its own build.docker.file",
				prev.app, name, dockerfileDisplayName(launch.Dockerfile),
				prev.artifact, artifact,
			)
		}
		owner[launch.Dockerfile] = dockerfileOwner{app: name, artifact: artifact}

		tags := []string{}
		// Only push to ecrRootRegion, images are replicated to other regions
		tag := fmt.Sprintf(
			"%s.dkr.ecr.%s.amazonaws.com/%s:%s",
			environment.ECRAccountID(), ecrRootRegion, artifact, environment.ShortSHA1(),
		)
		tags = append(tags, tag)

		targets[launch.Dockerfile] = DockerTarget{
			Tags:    tags,
			Command: launch.BuildCommand,
		}
	}
	return targets, artifacts, nil
}

// dockerfileDisplayName returns a human-readable Dockerfile name for
// error messages. An empty path is the default that docker.Build
// resolves to "Dockerfile".
func dockerfileDisplayName(path string) string {
	if path == "" {
		return "Dockerfile"
	}
	return path
}

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
// which only differ at runtime. A Kubernetes app pulls an image named
// after itself, so a Kubernetes app that shares another artifact also
// gets the shared image pushed under its own name.
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
		targets = map[string]DockerTarget{}
		// artifactDockerfile maps each built artifact to its target key.
		artifactDockerfile = map[string]string{}
		owner              = map[string]dockerfileOwner{}
		artifacts          []*catapult.Artifact
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
		if dockerfile, ok := artifactDockerfile[artifact]; ok {
			fmt.Println(name, "shares artifact with", artifact)
			targets[dockerfile] = withAppTag(targets[dockerfile], name, launch)
			continue
		}
		artifactDockerfile[artifact] = launch.Dockerfile

		// Two apps publishing to different artifacts cannot share a
		// Dockerfile path: keyed by path, the second would overwrite the
		// first and one image would never ship. Fail loudly.
		if prev, ok := owner[launch.Dockerfile]; ok {
			// Lead with the diagnosis, the Dockerfile path, and the fix so the
			// message stays actionable even if CircleCI secret-masking blanks
			// the app or artifact names at the end (they can match APP_NAME).
			return nil, nil, fmt.Errorf(
				"shared Dockerfile path collision: %q is claimed by two apps that "+
					"publish to different artifacts, so goci cannot build distinct "+
					"images from it. Give each app its own build.docker.file. "+
					"Conflicting apps: %q (artifact %q) and %q (artifact %q)",
				dockerfileDisplayName(launch.Dockerfile),
				prev.app, prev.artifact, name, artifact,
			)
		}
		owner[launch.Dockerfile] = dockerfileOwner{app: name, artifact: artifact}

		targets[launch.Dockerfile] = withAppTag(DockerTarget{
			Tags:    []string{ecrTag(artifact)},
			Command: launch.BuildCommand,
		}, name, launch)
	}
	return targets, artifacts, nil
}

// withAppTag adds a tag under the app's own name when a Kubernetes app
// publishes to an artifact with a different name.
func withAppTag(target DockerTarget, name string, launch *repo.AppConfig) DockerTarget {
	if launch.IsKubernetes && name != launch.ArtifactName {
		target.Tags = append(target.Tags, ecrTag(name))
	}
	return target
}

// ecrTag returns the image tag for repository at the current commit. Only
// push to ecrRootRegion, images are replicated to other regions.
func ecrTag(repository string) string {
	return fmt.Sprintf(
		"%s.dkr.ecr.%s.amazonaws.com/%s:%s",
		environment.ECRAccountID(), ecrRootRegion, repository, environment.ShortSHA1(),
	)
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

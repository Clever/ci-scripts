package lambda

import (
	"fmt"
	"strings"

	"github.com/Clever/catapult/gen-go/models"
	"github.com/Clever/ci-scripts/internal/catapult"
	"github.com/Clever/ci-scripts/internal/environment"
	"github.com/Clever/ci-scripts/internal/repo"
)

// LambdaTarget represents the information needed to build and publish a
// lambda to S3.
type LambdaTarget struct {
	// Zip is the the path to where the lambda artifact will be located
	// on the local FS
	Zip string
	// Command is the command to run to build the lambda artifact
	Command string
}

// BuildTargets returns a set of lambda targets to build and publish to
// S3 as well as a list of artifacts to be published to catapult. The
// targets map has the artifact name as the key and a build command and
// destination zip file in the value struct. Any apps with a shared
// artifact will have only one entry in the map, but will still have
// individual entries in the catapult build artifacts
func BuildTargets(apps map[string]*models.LaunchConfig) (map[string]LambdaTarget, []*catapult.Artifact) {
	var (
		targets   = map[string]LambdaTarget{}
		done      = map[string]struct{}{}
		artifacts []*catapult.Artifact
	)

	for name, launch := range apps {
		if !repo.IsLambdaRunType(launch) {
			continue
		}

		artifact := repo.ArtifactName(name, launch)
		artifacts = append(artifacts, &catapult.Artifact{
			RunType:   string(models.RunTypeLambda),
			ID:        name,
			Branch:    environment.Branch(),
			Source:    fmt.Sprintf("github:Clever/%s@%s", environment.Repo(), environment.FullSHA1()),
			Artifacts: fmt.Sprintf("lambda:clever/%s@%s;S3Key=\"%s,%s", artifact, environment.ShortSHA1(), s3Key(artifact), s3Buckets()),
		})

		if _, ok := done[artifact]; ok {
			fmt.Println(name, "shares artifact with", artifact)
			continue
		}
		done[artifact] = struct{}{}
		targets[artifact] = LambdaTarget{
			Zip:     fmt.Sprintf("./bin/%s.zip", artifact),
			Command: repo.BuildCommand(launch),
		}
	}
	return targets, artifacts
}

func s3Key(artifactName string) string {
	return fmt.Sprintf("%[1]s/%[2]s/%[1]s.zip", artifactName, environment.ShortSHA1())
}

func s3Buckets() string {
	out := []string{}
	for _, r := range environment.LambdaRegions {
		out = append(out, fmt.Sprintf("S3Buckets={%[1]s=\"%[2]s-%[1]s", r, environment.LambdaArtifactBucketPrefix()))
	}
	return strings.Join(out, ",")
}

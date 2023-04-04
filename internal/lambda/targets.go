package lambda

import (
	"fmt"
	"strings"

	"github.com/Clever/catapult/gen-go/models"
	"github.com/Clever/ci-scripts/internal/catapult"
	"github.com/Clever/ci-scripts/internal/environment"
	"github.com/Clever/ci-scripts/internal/repo"
)

// BuildTargets returns a set of lambda targets to build and publish to
// S3 as well as a list of artifacts to be published to catapult. The
// targets map has the artifact name as the key and the already built
// local archive location as the value. Any apps with a shared artifact
// will have only one entry in the map, but will still have individual
// entries in the catapult build artifacts
func BuildTargets(apps map[string]*models.LaunchConfig) (map[string]string, []*catapult.Artifact) {
	var (
		targets   = map[string]string{}
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
			Branch:    environment.Branch,
			Source:    fmt.Sprintf("github:Clever/%s@%s", environment.Repo, environment.FullSHA1),
			Artifacts: fmt.Sprintf("lambda:clever/%s@%s;S3Key=\\\"%s,%s", artifact, environment.ShortSHA1, s3Key(artifact), s3Buckets()),
		})

		if _, ok := done[artifact]; ok {
			fmt.Println(name, "shares artifact with", artifact)
			continue
		}
		done[artifact] = struct{}{}
		// Right now we aren't yet building source code into zips in
		// this application so we will just pull the assumed built file
		// name from the existing CI script until we do handle this.
		targets[artifact] = fmt.Sprintf("./bin/%s.zip", name)
	}
	return targets, artifacts
}

func s3Key(artifactName string) string {
	return fmt.Sprintf("%[1]s/%[2]s/%[1]s.zip", artifactName, environment.ShortSHA1)
}

func s3Buckets() string {
	out := []string{}
	for _, r := range environment.Regions {
		out = append(out, fmt.Sprintf("S3Buckets={%[1]s=\\\"%[2]s-%[1]s", r, environment.LambdaArtifactBucketPrefix))
	}
	return strings.Join(out, ",")
}

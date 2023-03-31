package lambda

import (
	"fmt"

	"github.com/Clever/catapult/gen-go/models"
	"github.com/Clever/ci-scripts/internal/repo"
)

// BuildTargets returns a map of lambda targets to publish with the
// artifact name as the key and the already built archive location as
// the value. Any apps with a shared artifact will have only one entry
// in the map.
func BuildTargets(apps map[string]*models.LaunchConfig) map[string]string {
	targets := map[string]string{}
	done := map[string]struct{}{}

	for name, launch := range apps {
		if !repo.IsLambdaRunType(launch) {
			continue
		}

		artifact := repo.ArtifactName(name, launch)
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
	return targets
}

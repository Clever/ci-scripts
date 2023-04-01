package catapult

import (
	"context"
	"fmt"

	"github.com/Clever/ci-scripts/internal/environment"
	"github.com/Clever/circle-ci-integrations/gen-go/client"
	"github.com/Clever/circle-ci-integrations/gen-go/models"
	"github.com/Clever/kayvee-go/v7/logger"
)

// Artifact aliases a catapult models.CatapultApplication, and contains
// information about the location of a build artifact so that catapult
// can correctly inject it at deploy time.
type Artifact = models.CatapultApplication

// Catapult wraps the circle-ci-integrations service with a trimmed down
// and simplified API.
type Catapult struct {
	client client.Client
}

// New initializes Catapult.
func New() (*Catapult, error) {
	cli, err := client.NewFromDiscovery(logger.NewConcreteLogger("goci"), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize circle-ci-integrations client: %v", err)
	}
	return &Catapult{client: cli}, nil
}

// Publish a list of build artifacts to catapult.
func (c *Catapult) Publish(ctx context.Context, artifacts []*Artifact) error {
	for _, art := range artifacts {
		err := c.client.PostCatapultV2(ctx, &models.CatapultPublishRequest{
			Username: environment.User,
			Reponame: environment.Repo,
			Buildnum: environment.CircleBuildNum,
			App:      art,
		})
		if err != nil {
			return fmt.Errorf("failed to publish %s with catapult: %v", art.ID, err)
		}
	}
	return nil
}

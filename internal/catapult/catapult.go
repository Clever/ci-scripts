package catapult

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Clever/ci-scripts/internal/environment"
	"github.com/Clever/circle-ci-integrations/gen-go/client"
	"github.com/Clever/circle-ci-integrations/gen-go/models"
	"github.com/Clever/ci-scripts/internal/service-util"

	"golang.org/x/sync/errgroup"
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

// New initializes Catapult with a circle-ci-integrations client that
// handles basic auth and discovers it's url via ci environment variables.
func New() *Catapult {
	// circle-ci-integrations up until this app was requested against in
	// ci via curl. Because of this the url environment variable was the
	// full protocol, hostname and path. This cleans up the variable so
	// we only have the proto and hostname. There are two separate
	// variables provided to provide legacy support so clean up both
	// possibilities
	url := strings.TrimSuffix(environment.CatapultURL(), "/v2/catapult")
	url = strings.TrimSuffix(url, "/catapult")
	var rt http.RoundTripper = &basicAuthTransport{}
	cli := client.New(url, serviceutil.FmtPrinlnLogger{}, &rt)
	cli.SetTimeout(15 * time.Second)
	return &Catapult{client: cli}
}

// SyncCatalogEntity syncs passed in entity to catalog-config by calling circle-ci-integrations
func (c *Catapult) SyncCatalogEntity(ctx context.Context, entity *models.SyncCatalogEntityInput) error {
	branch := environment.Branch()
	dryRun := branch != "master"
	entity.Branch = &branch
	entity.DryRun = &dryRun
	fmt.Printf("Syncing catalog entity %s with type %s on branch %s with dry run %t\n", entity.Entity, entity.Type, branch, dryRun)
	err := c.client.SyncCatalogEntity(ctx, entity)
	if err != nil {
		return fmt.Errorf("failed to sync catalog entity %s with catalogue config: %v", entity.Entity, err)
	}
	return nil
}

// Publish a list of build artifacts to catapult.
func (c *Catapult) Publish(ctx context.Context, artifacts []*Artifact) error {
	grp, grpCtx := errgroup.WithContext(ctx)

	for _, art := range artifacts {
		grp.Go(func() error {
			fmt.Println("Publishing", art.ID)
			err := c.client.PostCatapultV2(grpCtx, &models.CatapultPublishRequest{
				Username: environment.CircleUser(),
				Reponame: environment.Repo(),
				Buildnum: environment.CircleBuildNum(),
				App:      art,
			})
			if err != nil {
				return fmt.Errorf("failed to publish %s with catapult: %v", art.ID, err)
			}

			err = c.SyncCatalogEntity(grpCtx, &models.SyncCatalogEntityInput{
				Entity: art.ID,
				Type:   "application",
			})
			if err != nil {
				fmt.Println("failed to sync catalog app", art.ID, "with catalogue config:", err)
			}
			return nil
		})
	}
	return grp.Wait()
}

// Deploy a list of apps via catapult. Note that it is only possible to
// deploy to production, even if you pass an env param to
// circle-ci-integrations, which seems to ignore the param.
func (c *Catapult) Deploy(ctx context.Context, apps []string) error {
	for _, app := range apps {
		fmt.Println("Deploying", app)
		err := c.client.PostDapple(ctx, &models.DeployRequest{
			Appname:  app,
			Buildnum: environment.CircleBuildNum(),
			Reponame: environment.Repo(),
			Username: environment.CircleUser(),
		})
		if err != nil {
			return fmt.Errorf("failed to deploy %s: %v", app, err)
		}
	}
	return nil
}

// Wraps the default http transport in a very thin wrapper which just
// adds basic auth to all of the requests. The auth params are pulled
// from the ci environment.
type basicAuthTransport struct{}

func (ba *basicAuthTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r.SetBasicAuth(environment.CatapultUser(), environment.CatapultPassword())
	return http.DefaultTransport.RoundTrip(r)
}

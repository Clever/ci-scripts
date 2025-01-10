package catapult

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Clever/ci-scripts/internal/environment"
	"github.com/Clever/circle-ci-integrations/gen-go/client"
	"github.com/Clever/circle-ci-integrations/gen-go/models"
	"github.com/Clever/wag/logging/wagclientlogger"
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
	url := strings.TrimSuffix(environment.CatapultURL, "/v2/catapult")
	url = strings.TrimSuffix(url, "/catapult")
	var rt http.RoundTripper = &basicAuthTransport{}
	cli := client.New(url, fmtPrinlnLogger{}, &rt)
	return &Catapult{client: cli}
}

// Publish a list of build artifacts to catapult.
func (c *Catapult) Publish(ctx context.Context, artifacts []*Artifact) error {
	for _, art := range artifacts {
		err := c.client.PostCatapultV2(ctx, &models.CatapultPublishRequest{
			Username: environment.CircleUser,
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

func (c *Catapult) Deploy(ctx context.Context, apps []string) error {
	for _, app := range apps {
		fmt.Println("Deploying", app)
		err := c.client.PostDapple(ctx, &models.DeployRequest{
			Appname:     app,
			Buildnum:    environment.CircleBuildNum,
			Reponame:    environment.Repo,
			Username:    environment.CircleUser,
			Environment: "andru",
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
	r.SetBasicAuth(environment.CatapultUser, environment.CatapultPassword)
	return http.DefaultTransport.RoundTrip(r)
}

// A lightweight logger which prints the wag client logs to standard out.
type fmtPrinlnLogger struct{}

func (fmtPrinlnLogger) Log(level wagclientlogger.LogLevel, title string, data map[string]interface{}) {
	bs, _ := json.Marshal(data)
	fmt.Printf("%s - %s %s\n", levelString(level), title, string(bs))
}

func levelString(l wagclientlogger.LogLevel) string {
	switch l {
	case 0:
		return "TRACE"
	case 1:
		return "DEBUG"
	case 2:
		return "INFO"
	case 3:
		return "WARNING"
	case 4:
		return "ERROR"
	case 5:
		return "CRITICAL"
	case 6:
		return "FROM_ENV"
	default:
		return "INFO"
	}
}

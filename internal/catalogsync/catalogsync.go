package catalogsync

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Clever/ci-scripts/internal/environment"
	"github.com/Clever/ci-scripts/internal/logger"
	"github.com/Clever/circle-ci-integrations/gen-go/client"
	"github.com/Clever/circle-ci-integrations/gen-go/models"
)

type Client struct {
	client client.Client
}

func New() *Client {
	url := environment.CIIntegrationsUrl()
	var rt http.RoundTripper = &basicAuthTransport{}
	cli := client.New(url, logger.FmtPrinlnLogger{}, &rt)
	cli.SetTimeout(15 * time.Second)
	return &Client{client: cli}
}

func (c *Client) SyncEntity(ctx context.Context, entity *models.SyncCatalogEntityInput) error {
	branch := environment.Branch()
	dryRun := branch != "master"
	entity.Branch = &branch
	entity.DryRun = &dryRun

	fmt.Printf("Syncing catalog entity %s with type %s on branch %s with dry run %t\n", entity.Entity, entity.Type, branch, dryRun)
	if err := c.client.SyncCatalogEntity(ctx, entity); err != nil {
		return fmt.Errorf("failed to sync catalog entity %s: %v", entity.Entity, err)
	}
	return nil
}

type basicAuthTransport struct{}

func (ba *basicAuthTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r.SetBasicAuth(environment.CIIntegrationsUser(), environment.CIIntegrationsPassword())
	return http.DefaultTransport.RoundTrip(r)
}

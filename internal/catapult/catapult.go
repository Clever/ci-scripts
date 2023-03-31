package catapult

import (
	"context"

	"github.com/Clever/circle-ci-integrations/gen-go/client"
	"github.com/Clever/kayvee-go/v7/logger"
)

type Catapult struct {
	client *client.Client
}

func New() *Catapult {
	// l := logger.New
	cli := client.NewFromDiscovery()
}

func (c *Catapult) Publish(ctx context.Context) error {
	cli := client.Client{}
}

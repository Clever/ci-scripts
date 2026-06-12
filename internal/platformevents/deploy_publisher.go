package platformevents

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Clever/ci-scripts/internal/environment"
	"github.com/Clever/ci-scripts/internal/platformevents/schemas/deploycreated"
	"github.com/Clever/ci-scripts/internal/repo"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
)

const (
	eventBridgeName  = "production--platform-events"
	deployDetailType = "deploy.created"
	source           = "circle-ci"
)

type DeployPublisher struct {
	client *eventbridge.Client
}


func NewDeployPublisher(ctx context.Context) *DeployPublisher {
	cfg := environment.AWSCfg(ctx, environment.OidcEventBridgeRole())
	return &DeployPublisher{
		client: eventbridge.NewFromConfig(cfg),
	}
}

func (d *DeployPublisher) DeployApps(ctx context.Context, apps []string) error {
	for _, app := range apps {
		envs, err := repo.AutoDeployEnvs(app)
		if err != nil {
			return err
		}
		for _, env := range envs {
			err := d.deployApp(ctx, app, env)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (d *DeployPublisher) deployApp(ctx context.Context, app, env string) error {
	buildID := environment.ShortSHA1()
	repoName := environment.Repo()
	githubUser := environment.CircleTriggeredBy()
	clusterEnvironment := getClusterEnvironment(env)

	fmt.Println("Deploying", app, "to", env, "with build ID", buildID)

	event := deploycreated.Detail{
		App:                app,
		Repo:               repoName,
		User:               deploycreated.User{GithubUsername: strPtr(githubUser)},
		Environment:        env,
		TargetRevision:     buildID,
		ClusterEnvironment: clusterEnvironment,
	}

	detail, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal deploy event: %w", err)
	}

	_, err = d.client.PutEvents(ctx, &eventbridge.PutEventsInput{
		Entries: []types.PutEventsRequestEntry{
			{
				EventBusName: strPtr(eventBridgeName),
				DetailType:   strPtr(deployDetailType),
				Source:       strPtr(source),
				Detail:       strPtr(string(detail)),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to put event to EventBridge: %w", err)
	}

	return nil
}

func strPtr(s string) *string {
	return &s
}

func getClusterEnvironment(env string) string {
	if strings.Contains(env, "production") {
		return "production"
	}
	return "development"
}

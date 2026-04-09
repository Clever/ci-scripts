package slingshot

import (
	"context"
	"fmt"
	"strings"

	"github.com/Clever/ci-scripts/internal/environment"
	slingshotModels "github.com/Clever/slingshot/gen-go/models"
	slingshotClient "github.com/Clever/slingshot/gen-go/client"
	"github.com/Clever/ci-scripts/internal/logger"
	"github.com/Clever/ci-scripts/internal/repo"
)

type Slingshot struct {
	client slingshotClient.Client
}

func New() *Slingshot {
	return &Slingshot{
		client: slingshotClient.New(environment.SlingshotURL(), logger.FmtPrinlnLogger{}, nil),
	}
}

func (s *Slingshot) DeployApps(ctx context.Context, apps []string) error {
	for _, app := range apps {
		envs, err := repo.AutoDeployEnvs(app)
		if err != nil {
			return err
		}
		for _, env := range envs {
			err := s.deployApp(ctx, app, env)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Slingshot) deployApp(ctx context.Context, app, env string) error {
	buildID := environment.ShortSHA1()
	repoName := environment.Repo()
	githubUser := environment.CircleTriggeredBy()
	clusterEnvironment := getSlingshotClusterEnvironment(env)

	fmt.Println("Deploying", app, "to", env, "with build ID", buildID)

	return s.client.CreateDeploymentArtifact(ctx, &slingshotModels.CreateDeploymentArtifactRequest{
		App:                &app,
		ClusterEnvironment: &clusterEnvironment,
		Environment:        &env,
		Repo:               &repoName,
		TargetRevision:     &buildID,
		User: &slingshotModels.User{
			GithubUsername: githubUser,
		},
	})
}

func getSlingshotClusterEnvironment(env string) string {
	if strings.Contains(env, "production") {
		return "production"
	}
	return "development"
}

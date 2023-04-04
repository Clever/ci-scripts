package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"

	"github.com/Clever/catapult/gen-go/models"
	"github.com/Clever/ci-scripts/internal/catapult"
	"github.com/Clever/ci-scripts/internal/docker"
	"github.com/Clever/ci-scripts/internal/environment"
	"github.com/Clever/ci-scripts/internal/lambda"
	"github.com/Clever/ci-scripts/internal/repo"
)

// This app assumes the code has been checked out and that the
// repository is the working directory.

func main() {
	if err := run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run() error {
	apps, err := repo.DiscoverApplications("./launch")
	if err != nil {
		return err
	}

	var (
		ctx       = context.Background()
		cfg       aws.Config
		artifacts []*catapult.Artifact
	)

	cfg, err = awsCfg(ctx)
	if err != nil {
		return err
	}

	dockerTargets, dockerArtifacts := docker.BuildTargets(apps)
	lambdaTargets, lambdaArtifacts := lambda.BuildTargets(apps)
	artifacts = append(artifacts, dockerArtifacts...)
	artifacts = append(artifacts, lambdaArtifacts...)
	// We don't handle all application types yet (e.g. spark), so error out
	// instead of silently not building everything.
	if err = allAppsBuilt(apps, artifacts); err != nil {
		return err
	}

	if len(dockerTargets) > 0 {
		dkr, err := docker.New(ctx, cfg)
		if err != nil {
			return err
		}

		for dockerfile, tags := range dockerTargets {
			if err = dkr.Build(ctx, ".", dockerfile, tags); err != nil {
				return err
			}
			if err = dkr.Push(ctx, tags); err != nil {
				return err
			}
		}
	}

	if len(lambdaTargets) > 0 {
		lmda := lambda.New(cfg)

		for artifact, binary := range lambdaTargets {
			if err := lmda.Publish(ctx, binary, artifact); err != nil {
				return err
			}
		}
	}
	return catapult.New().Publish(ctx, artifacts)
}

// allAppsBuilt returns an error if any apps are missing a build artifact.
func allAppsBuilt(discoveredApps map[string]*models.LaunchConfig, builtApps []*catapult.Artifact) error {
	if len(discoveredApps) == len(builtApps) {
		return nil
	}

	missing := []string{}
	for name := range discoveredApps {
		found := false
		for _, b := range builtApps {
			if name == b.ID {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, name)
		}
	}
	return fmt.Errorf("applications %s not built", strings.Join(missing, ", "))
}

func awsCfg(ctx context.Context) (aws.Config, error) {
	opts := []func(*config.LoadOptions) error{
		config.WithRegion("us-west-1"),
	}

	// In local environment we use the default credentials chain that
	// will automatically pull creds from saml2aws,
	if !environment.Local {
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(environment.ECRAccessKeyID, environment.ECRSecretAccessKey, ""),
		))
	}

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to load aws config: %v", err)
	}

	return cfg, nil
}

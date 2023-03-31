package main

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/sync/errgroup"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"

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

	ctx := context.Background()
	cfg, err := awsCfg(ctx)
	if err != nil {
		return err
	}

	dockerTargets := docker.BuildTargets(apps)
	if len(dockerTargets) > 0 {
		dkr, err := docker.New(ctx, cfg)
		if err != nil {
			return err
		}

		for dockerfile, tags := range dockerTargets {
			if err := dkr.Build(ctx, ".", dockerfile, tags); err != nil {
				return err
			}

			// Take advantage of concurrency to speed up this multi-push
			// until we enable replication.
			grp, grpCtx := errgroup.WithContext(ctx)
			for _, tag := range tags {
				tag := tag
				grp.Go(func() error { return dkr.Push(grpCtx, tag) })
			}

			if err := grp.Wait(); err != nil {
				return err
			}
		}
	}

	lambdaTargets := lambda.BuildTargets(apps)
	if len(lambdaTargets) > 0 {
		lmda := lambda.New(cfg)

		grp, grpCtx := errgroup.WithContext(ctx)
		for artifact, binary := range lambdaTargets {
			artifact := artifact
			binary := binary

			for _, region := range environment.AWSRegions {
				region := region
				grp.Go(func() error { return lmda.Publish(grpCtx, binary, artifact, region) })
			}
		}

		if err := grp.Wait(); err != nil {
			return err
		}
	}

	// TODO: publish catapult
	return nil
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

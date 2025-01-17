package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Clever/catapult/gen-go/models"
	"github.com/Clever/ci-scripts/internal/catapult"
	"github.com/Clever/ci-scripts/internal/docker"
	"github.com/Clever/ci-scripts/internal/environment"
	"github.com/Clever/ci-scripts/internal/lambda"
	"github.com/Clever/ci-scripts/internal/repo"
)

const usage = "usage: goci <detect|artifact-build-publish-deploy>"

// This app assumes the code has been checked out and that the
// repository is the working directory.

func main() {
	if len(os.Args) < 2 {
		fmt.Println("requires 1 argument.", usage)
		os.Exit(1)
	}
	mode := os.Args[1]
	if err := run(mode); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run(mode string) error {
	if strings.Contains(environment.Branch, "/") {
		return fmt.Errorf("branch name %s contains a `/` character, which is not supported by catapult", environment.Branch)
	}

	apps, err := repo.DiscoverApplications("./launch")
	if err != nil {
		return err
	}

	appIDs := []string{}
	for app := range apps {
		appIDs = append(appIDs, app)
	}

	switch mode {
	case "detect":
		fmt.Println(strings.Join(appIDs, " "))
		return nil
	case "artifact-build-publish-deploy":
		// continue
	default:
		return fmt.Errorf("unknown mode %s. %s", mode, usage)
	}

	if len(apps) == 0 {
		fmt.Println("No applications have buildable changes. If this is unexpected, " +
			"double check your artifact dependency configuration in the launch yaml.")
		return nil
	}

	var (
		ctx       = context.Background()
		artifacts []*catapult.Artifact
	)

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
		dkr, err := docker.New(ctx)
		if err != nil {
			return err
		}

		for dockerfile, t := range dockerTargets {
			if err = repo.ExecBuild(t.Command); err != nil {
				return err
			}

			if err = dkr.Build(ctx, ".", dockerfile, t.Tags); err != nil {
				return err
			}
			if err = dkr.Push(ctx, t.Tags); err != nil {
				return err
			}
		}
	}

	if len(lambdaTargets) > 0 {
		lmda := lambda.New(ctx)

		for artifact, t := range lambdaTargets {
			if err = repo.ExecBuild(t.Command); err != nil {
				return err
			}

			if err = lmda.Publish(ctx, t.Zip, artifact); err != nil {
				return err
			}
		}
	}
	cp := catapult.New()

	if err = cp.Publish(ctx, artifacts); err != nil {
		return err
	}

	if environment.Branch == "master" {
		return cp.Deploy(ctx, appIDs)
	}
	return nil
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

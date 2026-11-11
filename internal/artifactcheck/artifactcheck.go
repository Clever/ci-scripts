package artifactcheck

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Clever/ci-scripts/internal/environment"
	"github.com/Clever/ci-scripts/internal/lambda"
	"github.com/Clever/ci-scripts/internal/repo"
)

const lambdaArtifactRegion = "us-west-2"

type ImageChecker interface {
	ImageExists(ctx context.Context, repository, tag string) (bool, error)
}

type ObjectChecker interface {
	ObjectExists(ctx context.Context, bucket, key string) (bool, error)
}

// FilterDeployable returns the appIDs whose artifacts exist. A missing
// artifact for an app with dependencies fails the check; a missing artifact
// for an always-build app is skipped with a warning.
func FilterDeployable(ctx context.Context, apps map[string]*repo.AppConfig, appIDs []string, images ImageChecker, objects ObjectChecker) ([]string, error) {
	var (
		deployable []string
		failures   []string
	)
	for _, appID := range appIDs {
		app := apps[appID]
		if !app.IsKubernetes {
			deployable = append(deployable, appID)
			continue
		}

		reference, exists, err := checkArtifact(ctx, appID, app, images, objects)
		if err != nil {
			return nil, fmt.Errorf("failed to check artifact for app %s (%s): %w", appID, reference, err)
		}
		switch {
		case exists:
			deployable = append(deployable, appID)
		case len(app.Dependencies) > 0:
			failures = append(failures, fmt.Sprintf("app %s at SHA %s is missing %s", appID, environment.ShortSHA1(), reference))
		default:
			fmt.Printf("WARNING: skipping deploy of app %s at SHA %s: %s is missing\n", appID, environment.ShortSHA1(), reference)
		}
	}
	if len(failures) > 0 {
		return nil, errors.New("artifacts missing for apps with changed dependencies:\n" + strings.Join(failures, "\n"))
	}
	return deployable, nil
}

func checkArtifact(ctx context.Context, appID string, app *repo.AppConfig, images ImageChecker, objects ObjectChecker) (string, bool, error) {
	if repo.IsLambdaRunType(app) {
		bucket := fmt.Sprintf("%s-%s", environment.LambdaArtifactBucketPrefix(), lambdaArtifactRegion)
		key := lambda.S3Key(app.ArtifactName)
		exists, err := objects.ObjectExists(ctx, bucket, key)
		return fmt.Sprintf("S3 object s3://%s/%s", bucket, key), exists, err
	}

	tag := environment.ShortSHA1()
	exists, err := images.ImageExists(ctx, appID, tag)
	return fmt.Sprintf("ECR image %s:%s", appID, tag), exists, err
}

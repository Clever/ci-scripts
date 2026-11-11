package artifactcheck_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Clever/ci-scripts/internal/artifactcheck"
	"github.com/Clever/ci-scripts/internal/artifactcheck/mocks"
	"github.com/Clever/ci-scripts/internal/repo"
)

const (
	testBucket    = "test-bucket-us-west-2"
	dockerPresent = "docker-app:abcdef1"
	lambdaPresent = testBucket + "/lambda-artifact/abcdef1/lambda-artifact.zip"
)

func setTestEnvironment(t *testing.T) {
	t.Setenv("CIRCLE_SHA1", "abcdef1234567890")
	t.Setenv("LAMBDA_AWS_BUCKET", "test-bucket")
	t.Setenv("ECR_ACCOUNT_ID", "123456789012")
}

func dockerApp(dependencies ...string) *repo.AppConfig {
	return &repo.AppConfig{RunType: repo.RunTypeDocker, ArtifactName: "docker-app", IsKubernetes: true, Dependencies: dependencies}
}

func lambdaApp(dependencies ...string) *repo.AppConfig {
	return &repo.AppConfig{RunType: repo.RunTypeLambda, ArtifactName: "lambda-artifact", IsKubernetes: true, Dependencies: dependencies}
}

func TestFilterDeployable(t *testing.T) {
	tests := []struct {
		name             string
		apps             map[string]*repo.AppConfig
		existingImages   map[string]bool
		existingObjects  map[string]bool
		expectedDeploy   []string
		expectedErrorHas []string
		expectedImages   []string
		expectedObjects  []string
	}{
		{
			name:            "present docker app deploys",
			apps:            map[string]*repo.AppConfig{"docker-app": dockerApp("src")},
			existingImages:  map[string]bool{dockerPresent: true},
			expectedDeploy:  []string{"docker-app"},
			expectedImages:  []string{dockerPresent},
			expectedObjects: nil,
		},
		{
			name:            "present lambda app deploys",
			apps:            map[string]*repo.AppConfig{"lambda-app": lambdaApp("src")},
			existingObjects: map[string]bool{lambdaPresent: true},
			expectedDeploy:  []string{"lambda-app"},
			expectedObjects: []string{lambdaPresent},
		},
		{
			name:             "missing docker image with dependencies fails naming app, SHA and tag",
			apps:             map[string]*repo.AppConfig{"docker-app": dockerApp("src")},
			expectedErrorHas: []string{"docker-app", "abcdef1", "docker-app:abcdef1"},
			expectedImages:   []string{dockerPresent},
		},
		{
			name:             "missing lambda zip with dependencies fails naming key",
			apps:             map[string]*repo.AppConfig{"lambda-app": lambdaApp("src")},
			expectedErrorHas: []string{"lambda-app", "abcdef1", "lambda-artifact/abcdef1/lambda-artifact.zip"},
			expectedObjects:  []string{lambdaPresent},
		},
		{
			name:           "missing docker image with empty dependencies is skipped",
			apps:           map[string]*repo.AppConfig{"docker-app": dockerApp()},
			expectedDeploy: nil,
			expectedImages: []string{dockerPresent},
		},
		{
			name:            "missing lambda zip with empty dependencies is skipped",
			apps:            map[string]*repo.AppConfig{"lambda-app": lambdaApp()},
			expectedObjects: []string{lambdaPresent},
		},
		{
			name: "catapult-only app is not checked",
			apps: map[string]*repo.AppConfig{
				"catapult-app": {RunType: repo.RunTypeDocker, ArtifactName: "catapult-app", Dependencies: []string{"src"}},
			},
			expectedDeploy: []string{"catapult-app"},
		},
		{
			name: "mixed set fails listing only the failure",
			apps: map[string]*repo.AppConfig{
				"docker-app":  dockerApp("src"),
				"lambda-app":  lambdaApp(),
				"present-app": {RunType: repo.RunTypeDocker, ArtifactName: "present-app", IsKubernetes: true, Dependencies: []string{"src"}},
			},
			existingImages:   map[string]bool{"present-app:abcdef1": true},
			expectedErrorHas: []string{"docker-app:abcdef1"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setTestEnvironment(t)
			images := &mocks.FakeImageChecker{Existing: test.existingImages}
			objects := &mocks.FakeObjectChecker{Existing: test.existingObjects}
			appIDs := make([]string, 0, len(test.apps))
			for appID := range test.apps {
				appIDs = append(appIDs, appID)
			}

			deployable, err := artifactcheck.FilterDeployable(context.Background(), test.apps, appIDs, images, objects)

			if test.expectedErrorHas != nil {
				require.Error(t, err)
				for _, fragment := range test.expectedErrorHas {
					assert.Contains(t, err.Error(), fragment)
				}
				return
			}
			require.NoError(t, err)
			assert.ElementsMatch(t, test.expectedDeploy, deployable)
			if test.expectedImages != nil {
				assert.ElementsMatch(t, test.expectedImages, images.Checked)
			} else {
				assert.Empty(t, images.Checked)
			}
			if test.expectedObjects != nil {
				assert.ElementsMatch(t, test.expectedObjects, objects.Checked)
			} else {
				assert.Empty(t, objects.Checked)
			}
		})
	}
}

func TestFilterDeployableMixedSetDoesNotListSkippedApp(t *testing.T) {
	setTestEnvironment(t)
	apps := map[string]*repo.AppConfig{"docker-app": dockerApp("src"), "lambda-app": lambdaApp()}

	_, err := artifactcheck.FilterDeployable(
		context.Background(), apps, []string{"docker-app", "lambda-app"},
		&mocks.FakeImageChecker{}, &mocks.FakeObjectChecker{},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "docker-app")
	assert.NotContains(t, err.Error(), "lambda-app")
}

func TestFilterDeployableReturnsCheckerError(t *testing.T) {
	setTestEnvironment(t)
	apps := map[string]*repo.AppConfig{"docker-app": dockerApp("src")}
	images := &mocks.FakeImageChecker{Error: assert.AnError}

	_, err := artifactcheck.FilterDeployable(context.Background(), apps, []string{"docker-app"}, images, &mocks.FakeObjectChecker{})

	require.ErrorIs(t, err, assert.AnError)
}

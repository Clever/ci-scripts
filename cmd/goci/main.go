package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/mod/modfile"

	"github.com/Clever/catapult/gen-go/models"
	"github.com/Clever/ci-scripts/internal/catapult"
	"github.com/Clever/ci-scripts/internal/docker"
	"github.com/Clever/ci-scripts/internal/environment"
	"github.com/Clever/ci-scripts/internal/lambda"
	"github.com/Clever/ci-scripts/internal/repo"
)

const usage = "usage: goci <validate|detect|artifact-build-publish-deploy>"

// This app assumes the code has been checked out and that the
// repository is the working directory.

// ValidationError represents an error that occurs during validation.
type ValidationError struct {
    Message string
}

func (e *ValidationError) Error() string {
    return e.Message
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("requires 1 argument.", usage)
		os.Exit(1)
	}
	mode := os.Args[1]
	if err := run(mode); err != nil {
		if _, ok := err.(*ValidationError); ok {
            fmt.Println("Validation error:", err)
            os.Exit(2) // Use a different exit code for validation errors
        } else {
            fmt.Println("Error:", err)
            os.Exit(1)
        }
	}
}

func run(mode string) error {
	apps, err := repo.DiscoverApplications("./launch")
	if err != nil {
		return err
	}

	appIDs := []string{}
	for app := range apps {
		appIDs = append(appIDs, app)
	}

	switch mode {
	case "validate":
		err := validateRun()
		if err != nil {
			return err
		}
		return nil
	case "detect":
		fmt.Println(strings.Join(appIDs, " "))
		return nil
	case "artifact-build-publish-deploy":
		// continue
	default:
		return fmt.Errorf("unknown mode %s. %s", mode, usage)
	}

	// We want to validate on every run, not just when the mode is "validate".
	if err = validateRun(); err != nil {
		return err
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
		dkr, err := docker.New(ctx, environment.OidcEcrUploadRole())
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
		lmda := lambda.New(ctx, environment.LambdaArtifactBucketPrefix())

		for artifact, t := range lambdaTargets {
			if err = repo.ExecBuild(t.Command); err != nil {
				return err
			}

			if err = lmda.Publish(ctx, t.Zip, artifact); err != nil {
				return err
			}
		}
	}
	cp := catapult.New(environment.CatapultURL(), environment.CircleUser(), environment.Repo(), environment.CircleBuildNum())

	if err = cp.Publish(ctx, artifacts, ); err != nil {
		return err
	}

	if environment.Branch() == "master" {
		return cp.Deploy(ctx, appIDs)
	}
	return nil
}

// validateRun checks the env.branch and go version to ensure the build is valid.
func validateRun() error {
	if strings.Contains(environment.Branch(), "/") {
        return &ValidationError{Message: fmt.Sprintf("branch name %s contains a `/` character, which is not supported by catapult", environment.Branch())}
	}

	latestGoVersion, err := fetchLatestGoVersion()
	if err != nil {
		return fmt.Errorf("failed to fetch latest Go version: %v", err)
	}

	goModPath := "./go.mod"
    fileBytes, err := os.ReadFile(goModPath)
    if err != nil {
        return fmt.Errorf("failed to read go.mod file: %v", err)
    }

	f, err := modfile.Parse("./go.mod", fileBytes , nil)
	if err != nil {
		return fmt.Errorf("failed to parse go.mod file: %v", err)
	}

	// trim the patch value from the authoring repositories go version - if 2 dots are present
	var trimmedVersion string
	if strings.Count(f.Go.Version, ".") == 2 {
		trimmedVersion = f.Go.Version[:len(f.Go.Version)-2]
	} else {
		trimmedVersion = f.Go.Version
	}

	version, e := strconv.ParseFloat(trimmedVersion, 64)

	if e != nil {
		return fmt.Errorf("failed to parse go version: %v", e)
	}

	// We will begin enforcing this policy for go version 1.24 and above, for now set the minimum version to 1.23
	if version <= 1.23 {
		version = 1.23
	}

	// trim the patch value from the latest go version
	latestGoVersion = latestGoVersion[:len(latestGoVersion)-2]
	newestGoVersion, e := strconv.ParseFloat(latestGoVersion, 64)
	if e != nil {
		return fmt.Errorf("failed to parse go version: %v", e)
	}

	if version < newestGoVersion - 0.01 {
        return &ValidationError{Message: fmt.Sprintf("go version %v is no longer supported. Please upgrade to version %v", version, newestGoVersion)}
	} else if version == newestGoVersion - 0.01 {
		// We'll give a PR comment to the Author to warn them about the need to upgrade
		fmt.Printf("Warning: This go version (%v) is nearing deprecation. Please upgrade to version %v\n", version, newestGoVersion)
	}

	return nil
}

// fetchLatestGoVersion fetches the latest Go version from the official Go download page.
func fetchLatestGoVersion() (string, error) {
	// official Go download page
    resp, err := http.Get("https://go.dev/dl/")
    if err != nil {
        return "", fmt.Errorf("failed to fetch Go download page: %v", err)
    }
    defer resp.Body.Close()
	
    if resp.StatusCode != http.StatusOK {
        return "", fmt.Errorf("failed to fetch Go download page: status code %d", resp.StatusCode)
    }

    bodyBytes, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", fmt.Errorf("failed to read response body: %v", err)
    }
    body := string(bodyBytes)

    // Extract the latest macOS (darwin) download URL
    re := regexp.MustCompile(`/dl/go[0-9]+\.[0-9]+\.[0-9]+\.darwin-arm64.pkg`)
    matches := re.FindStringSubmatch(body)
    if len(matches) == 0 {
        return "", fmt.Errorf("failed to find Go download URL")
    }
    goURL := "https://go.dev" + matches[0]

    // Extract the Go version number from the URL
    reVersion := regexp.MustCompile(`[0-9]+\.[0-9]+\.[0-9]+`)
    versionMatches := reVersion.FindStringSubmatch(goURL)
    if len(versionMatches) == 0 {
        return "", fmt.Errorf("failed to find Go version in URL")
    }
    goVersion := versionMatches[0]

    return goVersion, nil
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
package repo

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/ghodss/yaml"

	"github.com/Clever/catapult/gen-go/models"
)

// DiscoverApplications finds any launch config files in the specified
// directory and returns a map with the application name as the key and
// the corresponding launch config file as the value. DB launch configs
// are ignored. Any applications that do not have changes detected
// according to the launch config dependencies are filtered from the
// result set.
func DiscoverApplications(dir string) (map[string]*models.LaunchConfig, error) {
	fe, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("directory %s not found: %w", dir, err)
		}
		return nil, fmt.Errorf("failed to read launch directory: %v", err)
	}

	m := map[string]*models.LaunchConfig{}
	for _, f := range fe {
		if f.IsDir() {
			continue
		}
		if path.Ext(f.Name()) != ".yml" {
			continue
		}

		bs, err := os.ReadFile(path.Join(dir, f.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %v", f.Name(), err)
		}

		lc := models.LaunchConfig{}
		if err := yaml.Unmarshal(bs, &lc); err != nil {
			return nil, fmt.Errorf("failed to unmarshal yaml in %s: %v", f.Name(), err)
		}

		// These are DB launch configs, which we don't want to build.
		if lc.PodConfig == nil || lc.PodConfig.Group == "" {
			continue
		}

		// if changed, err := DetectArtifactDependencyChange(&lc); err != nil {
		// 	return nil, fmt.Errorf("failed to detect artifact dependency change for %s: %v", f.Name(), err)
		// } else if !changed {
		// 	continue
		// }

		m[strings.TrimSuffix(f.Name(), ".yml")] = &lc
	}

	return m, nil
}

// Dockerfile returns the dockerfile name specified in the launch config
// if any is present, otherwise it returns an empty string.
func Dockerfile(lc *models.LaunchConfig) string {
	if lc.Build != nil && lc.Build.Docker != nil {
		return lc.Build.Docker.File
	}
	return ""
}

// IsDockerRunType returns true if the launch config specifies a run
// type of docker.
func IsDockerRunType(lc *models.LaunchConfig) bool {
	if r := lc.Run; r != nil {
		switch r.Type {
		case models.RunTypeDocker:
			return true
		// for legacy support reasons, an empty run type is treated as a
		// run type of docker.
		case "":
			return true
		default:
			return false
		}
	}
	// no run object also counts as docker.
	return true
}

// IsLambdaRunType returns true if the launch config specifies a run
// type of lambda.
func IsLambdaRunType(lc *models.LaunchConfig) bool {
	if r := lc.Run; r != nil {
		switch r.Type {
		case models.RunTypeLambda:
			return true
		default:
			return false
		}
	}
	return false
}

// ArtifactName returns the correct artifact name for the application.
// The default pattern is the app name. There is an optional launch
// config override in order to enable sharing one artifact between
// multiple applications. This may happen with for example, sso and
// non-sso, where the application is the same or only differs at run
// time based on environmental configuration.
func ArtifactName(appName string, lc *models.LaunchConfig) string {
	artifactName := appName
	if lc.Build != nil && lc.Build.Artifact != nil && lc.Build.Artifact.Name != "" {
		artifactName = lc.Build.Artifact.Name
	}
	return artifactName
}

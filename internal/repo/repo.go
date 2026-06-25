package repo

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/ghodss/yaml"

	"github.com/Clever/catapult/gen-go/models"
)

// DiscoverApplications returns buildable apps with detected changes. It reads
// from both ./config (Kubernetes) and ./launch (Catapult), merging results with
// the Kubernetes config winning when the same app appears in both.
func DiscoverApplications() (map[string]*AppConfig, error) {
	catapultApps, catapultErr := discoverCatapultApplications("./launch")
	if catapultErr != nil && !errors.Is(catapultErr, os.ErrNotExist) {
		return nil, catapultErr
	}
	fmt.Println("Discovered Catapult apps:", len(catapultApps))
	

	kubernetesApps, kubernetesErr := discoverKubernetesApplications("./config")
	if kubernetesErr != nil && !errors.Is(kubernetesErr, os.ErrNotExist) {
		return nil, kubernetesErr
	}
	fmt.Println("Discovered Kubernetes apps:", len(kubernetesApps))

	if errors.Is(catapultErr, os.ErrNotExist) && errors.Is(kubernetesErr, os.ErrNotExist) {
		return nil, fmt.Errorf("no app configs found: expected a launch/ or config/ directory at the repo root")
	}

	apps := catapultApps
	if apps == nil {
		apps = map[string]*AppConfig{}
	}
	for name, kubernetesAC := range kubernetesApps {
		if catapultAC, exists := apps[name]; exists {
			if kubernetesAC.BuildCommand == "" && catapultAC.BuildCommand != "" {
				return nil, fmt.Errorf("%s has no build.command in config/%s/stack.yaml but one exists in launch/%s.yml — add build.command to stack.yaml to complete the migration", name, name, name)
			}
		}
		apps[name] = kubernetesAC
	}
	return apps, nil
}

// discoverCatapultApplications reads launch/*.yml files, skips DB configs, filters
// by change detection, and converts each to AppConfig.
func discoverCatapultApplications(dir string) (map[string]*AppConfig, error) {
	fe, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("directory %s not found: %w", dir, err)
		}
		return nil, fmt.Errorf("failed to read launch directory: %v", err)
	}

	apps := map[string]*AppConfig{}
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

		appName := strings.TrimSuffix(f.Name(), ".yml")
		ac := appConfigForCatapult(appName, &lc)
		if changed, err := DetectArtifactDependencyChange(ac.Dependencies); err != nil {
			return nil, fmt.Errorf("failed to detect artifact dependency change for %s: %v", f.Name(), err)
		} else if !changed {
			continue
		}

		apps[appName] = ac
	}
	return apps, nil
}

// discoverKubernetesApplications reads config/<app>/stack.yaml for each app
// directory under the given dir, filters out apps with no detected changes,
// and returns a map of app name to AppConfig.
func discoverKubernetesApplications(dir string) (map[string]*AppConfig, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("directory %s not found: %w", dir, err)
		}
		return nil, fmt.Errorf("failed to read config directory: %v", err)
	}

	apps := map[string]*AppConfig{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		appName := entry.Name()
		ac, err := appConfigForKubernetes(appName)
		if err != nil {
			return nil, fmt.Errorf("failed to read app config for %s: %v", appName, err)
		}

		changed, err := DetectArtifactDependencyChange(ac.Dependencies)
		if err != nil {
			return nil, fmt.Errorf("failed to detect artifact dependency change for %s: %v", appName, err)
		}
		if !changed {
			continue
		}

		apps[appName] = ac
	}
	return apps, nil
}

// IsDockerRunType returns true if the app should be built as a Docker image.
// An empty RunType defaults to docker.
func IsDockerRunType(ac *AppConfig) bool {
	return ac.RunType == RunTypeDocker || ac.RunType == ""
}

// IsLambdaRunType returns true if the app should be built as a Lambda artifact.
func IsLambdaRunType(ac *AppConfig) bool {
	return ac.RunType == RunTypeLambda
}

// ExecBuild runs the build command for the application artifact, if any
// exists. If the command is empty, it returns nil after performing nop.
func ExecBuild(c string) error {
	if c == "" {
		return nil
	}

	args := strings.Split(c, " ")
	cmd := exec.Command(args[0], args[1:]...)
	fmt.Println("Running build command:", cmd.String())

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run build command: %v", err)
	}
	fmt.Println("Build command completed successfully")

	return nil
}

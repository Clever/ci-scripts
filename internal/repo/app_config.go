package repo

import (
	"fmt"
	"os"

	"github.com/ghodss/yaml"

	"github.com/Clever/catapult/gen-go/models"
)

const (
	appStackConfigPath = "config/%s/stack.yaml"

	RunTypeDocker = "docker"
	RunTypeLambda = "lambda"
)

type appHelmYAML struct {
	Chart string `json:"chart"`
}

type appStackBlockYAML struct {
	Helm appHelmYAML `json:"helm"`
}

type appBuildYAML struct {
	Command      string   `json:"command"`
	ArtifactName string   `json:"artifactName"`
	Dockerfile   string   `json:"dockerfile"`
	Dependencies []string `json:"dependencies"`
}

type appStackYAML struct {
	AutoDeployEnvs []string          `json:"autoDeployEnvs"`
	Stack          appStackBlockYAML `json:"stack"`
	Build          appBuildYAML      `json:"build"`
}

// AppConfig holds build and runtime configuration
type AppConfig struct {
	Name            string
	RunType         string
	ArtifactName    string
	BuildCommand    string
	Dockerfile      string
	Dependencies    []string
	HasLaunchConfig bool
}

// ReadAppStackAutoDeployEnvs reads autoDeployEnvs from config/<app>/stack.yaml.
func AutoDeployEnvs(app string) ([]string, error) {
	path := fmt.Sprintf(appStackConfigPath, app)
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var stack appStackYAML
	if err := yaml.Unmarshal(b, &stack); err != nil {
		return nil, fmt.Errorf("failed to unmarshal stack.yaml for %s: %w", path, err)
	}
	if len(stack.AutoDeployEnvs) == 0 {
		return nil, nil
	}
	return stack.AutoDeployEnvs, nil
}

func appConfigForCatapult(appName string, lc *models.LaunchConfig) *AppConfig {
	runType := RunTypeDocker
	if lc.Run != nil && lc.Run.Type == models.RunTypeLambda {
		runType = RunTypeLambda
	}

	artifactName := appName
	var buildCommand, dockerfile string
	var dependencies []string
	if lc.Build != nil {
		if lc.Build.Artifact != nil {
			if lc.Build.Artifact.Name != "" {
				artifactName = lc.Build.Artifact.Name
			}
			buildCommand = lc.Build.Artifact.Command
			dependencies = lc.Build.Artifact.Dependencies
		}
		if lc.Build.Docker != nil {
			dockerfile = lc.Build.Docker.File
		}
	}

	return &AppConfig{
		Name:            appName,
		RunType:         runType,
		ArtifactName:    artifactName,
		BuildCommand:    buildCommand,
		Dockerfile:      dockerfile,
		Dependencies:    dependencies,
		HasLaunchConfig: true,
	}
}

func appConfigForKubernetes(appName string) (*AppConfig, error) {
	stackPath := fmt.Sprintf(appStackConfigPath, appName)
	b, err := os.ReadFile(stackPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read stack.yaml for %s: %w", appName, err)
	}
	var stack appStackYAML
	if err := yaml.Unmarshal(b, &stack); err != nil {
		return nil, fmt.Errorf("failed to unmarshal stack.yaml for %s: %w", stackPath, err)
	}

	runType := RunTypeDocker
	if stack.Stack.Helm.Chart == "clever-lambda" {
		runType = RunTypeLambda
	}

	artifactName := appName
	if stack.Build.ArtifactName != "" {
		artifactName = stack.Build.ArtifactName
	}

	return &AppConfig{
		Name:         appName,
		RunType:      runType,
		ArtifactName: artifactName,
		BuildCommand: stack.Build.Command,
		Dockerfile:   stack.Build.Dockerfile,
		Dependencies: stack.Build.Dependencies,

	}, nil
}

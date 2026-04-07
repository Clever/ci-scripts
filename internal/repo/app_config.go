package repo

import (
	"fmt"
	"os"

	"github.com/ghodss/yaml"
)

const (
	appStackConfigPath = "config/%s/stack.yaml"
	appValuesConfigPath = "config/%s/values.yaml"
)

type appStackYAML struct {
	AutoDeployEnvs []string `json:"autoDeployEnvs"`
}

type appValuesYAML struct {
	Run struct {
		Type string `json:"type"`
	} `json:"run"`
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

// AppType reads the run.type from config/<app>/values.yaml.
func AppType(app string) (string, error) {	
	path := fmt.Sprintf(appValuesConfigPath, app)
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	var values appValuesYAML
	if err := yaml.Unmarshal(b, &values); err != nil {
		return "", fmt.Errorf("failed to unmarshal values.yaml for %s: %w", path, err)
	}
	return values.Run.Type, nil
}

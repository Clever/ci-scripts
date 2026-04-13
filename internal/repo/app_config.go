package repo

import (
	"fmt"
	"os"

	"github.com/ghodss/yaml"
)

const (
	appStackConfigPath = "config/%s/stack.yaml"
)

type appStackYAML struct {
	AutoDeployEnvs []string `json:"autoDeployEnvs"`
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

package backstage

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// BackstageEntity represents a generic Backstage entity.
type Entity struct {
	APIVersion string                 `json:"apiVersion"` // The API version of the entity (e.g., "backstage.io/v1alpha1").
	Kind       string                 `json:"kind"`       // The kind of the entity (e.g., "Component", "API").
	Metadata   Metadata               `json:"metadata"`   // Metadata about the entity.
	Spec       map[string]interface{} `json:"spec"`       // The specification of the entity (varies by kind).
}

// BackstageMetadata represents the metadata of a Backstage entity.
type Metadata struct {
	Name        string            `json:"name"`                  // The name of the entity.
	Namespace   string            `json:"namespace,omitempty"`   // The namespace of the entity (optional).
	Description string            `json:"description,omitempty"` // A description of the entity (optional).
	Labels      map[string]string `json:"labels,omitempty"`      // Labels for the entity (optional).
	Annotations map[string]string `json:"annotations,omitempty"` // Annotations for the entity (optional).
	Tags        []string          `json:"tags,omitempty"`        // Tags for the entity (optional).
}

// GetEntityFromYaml reads a yaml file and returns a BackstageEntity.
func GetEntityFromYaml(filePath string) (*Entity, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %v", filePath, err)
	}
	var entity Entity
	err = yaml.Unmarshal(data, &entity)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal yaml: %v", err)
	}
	return &entity, nil
}

func (e *Entity) GetName() string {
	if e == nil {
		return ""
	}
	return e.Metadata.Name
}

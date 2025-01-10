package repo

import (
	"testing"

	"github.com/Clever/catapult/gen-go/models"
)

func TestDetectArtifactDependencyChange(t *testing.T) {
	// Uncomment this if you want to debug this function. Note that the
	// shell commands are running within this directory which limits
	// find from seeing all files in this repo.
	t.Skip("skipping test")
	lc := &models.LaunchConfig{
		Build: &models.LaunchBuild{
			Artifact: &models.BuildArtifact{
				Dependencies: []string{"*.go"},
			},
		},
	}

	changed, err := DetectArtifactDependencyChange(lc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Fatal("expected changed to be true")
	}
}

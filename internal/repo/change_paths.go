package repo

import (
	"fmt"
	"path"
	"path/filepath"
	"slices"
)

type ChangeSelection int

const (
	SelectByDependencies ChangeSelection = iota
	SelectByImportGraph
)

const defaultDockerfile = "Dockerfile"

// changePathFinder caches go inputs per package, because several apps can build from one package.
type changePathFinder struct {
	selection         ChangeSelection
	goInputsByPackage map[string][]string
}

func newChangePathFinder(selection ChangeSelection) *changePathFinder {
	return &changePathFinder{
		selection:         selection,
		goInputsByPackage: map[string][]string{},
	}
}

// paths returns the git pathspecs whose changes mark the app as changed.
func (finder *changePathFinder) paths(ac *AppConfig) ([]string, error) {
	if finder.selection == SelectByDependencies {
		return ac.Dependencies, nil
	}
	if ac.Package == "" {
		fmt.Printf("%s has no build.package in stack.yaml: selecting by build.dependencies only\n", ac.Name)
		return ac.Dependencies, nil
	}

	goPaths, err := finder.goInputs(ac.Package)
	if err != nil {
		return nil, err
	}
	paths := append(slices.Clone(goPaths), "go.mod", "go.sum", path.Join("config", ac.Name))
	if IsDockerRunType(ac) {
		paths = append(paths, dockerfilePath(ac))
	}
	return append(paths, ac.Dependencies...), nil
}

func (finder *changePathFinder) goInputs(packageDir string) ([]string, error) {
	key := filepath.Clean(packageDir)
	if paths, cached := finder.goInputsByPackage[key]; cached {
		return paths, nil
	}
	paths, err := goInputPaths(packageDir)
	if err != nil {
		return nil, err
	}
	finder.goInputsByPackage[key] = paths
	return paths, nil
}

func dockerfilePath(ac *AppConfig) string {
	if ac.Dockerfile == "" {
		return defaultDockerfile
	}
	return ac.Dockerfile
}

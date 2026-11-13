package repo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
)

// excludeTestFiles keeps _test.go edits in imported packages from selecting the app.
const excludeTestFiles = ":(exclude,glob)**/*_test.go"

type goListPackage struct {
	ImportPath string
	Dir        string
	Module     *goListModule
	EmbedFiles []string
	Error      *goListError
}

type goListModule struct {
	Main bool
}

type goListError struct {
	Err string
}

// goInputPaths returns pathspecs for the main module's Go packages and embedded files that packageDir builds from.
func goInputPaths(packageDir string) ([]string, error) {
	packages, err := listPackageDeps(packageDir)
	if err != nil {
		return nil, err
	}
	repoRoot, err := resolvedDir(".")
	if err != nil {
		return nil, err
	}

	paths := []string{excludeTestFiles}
	for _, pkg := range packages {
		if pkg.Error != nil {
			fmt.Printf("skipping %s in the import graph of %s: %s\n", pkg.ImportPath, packageDir, pkg.Error.Err)
			continue
		}
		if pkg.Module == nil || !pkg.Module.Main {
			continue
		}
		packagePaths, err := packageInputPaths(repoRoot, pkg)
		if err != nil {
			return nil, err
		}
		paths = append(paths, packagePaths...)
	}
	return paths, nil
}

// listPackageDeps tolerates broken dependencies, such as code that build.command generates, but not a broken main package.
func listPackageDeps(packageDir string) ([]goListPackage, error) {
	command := exec.Command("go", "list", "-e", "-deps", "-json=ImportPath,Dir,Module,EmbedFiles,Error", "./"+filepath.Clean(packageDir))
	command.Stderr = os.Stderr
	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("go list -deps %s: %v", packageDir, err)
	}

	packages := []goListPackage{}
	decoder := json.NewDecoder(bytes.NewReader(output))
	for decoder.More() {
		var pkg goListPackage
		if err := decoder.Decode(&pkg); err != nil {
			return nil, fmt.Errorf("decode go list output: %v", err)
		}
		packages = append(packages, pkg)
	}

	// go list -deps prints the named package last.
	mainPackage := packages[len(packages)-1]
	if mainPackage.Error != nil {
		return nil, fmt.Errorf("build.package %s: %s", packageDir, mainPackage.Error.Err)
	}
	return packages, nil
}

// packageInputPaths globs *.go per directory so files added or deleted in the compare range still match.
func packageInputPaths(repoRoot string, pkg goListPackage) ([]string, error) {
	packageDir, err := resolvedDir(pkg.Dir)
	if err != nil {
		return nil, err
	}
	relativeDir, err := filepath.Rel(repoRoot, packageDir)
	if err != nil {
		return nil, err
	}
	relativeDir = filepath.ToSlash(relativeDir)

	paths := []string{":(glob)" + path.Join(relativeDir, "*.go")}
	for _, embedFile := range pkg.EmbedFiles {
		paths = append(paths, path.Join(relativeDir, embedFile))
	}
	return paths, nil
}

func resolvedDir(dir string) (string, error) {
	absolute, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(absolute)
}

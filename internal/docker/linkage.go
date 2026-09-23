package docker

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
)

// linkageCheck runs inside the built image and resolves the entrypoint
// binary's shared libraries with ldd. It exits 1 only when the binary
// will not load; unsupported images (no ldd, non-ELF entrypoint) exit 0.
//
//go:embed linkage_check.sh
var linkageCheck string

// Linkage check modes, set via CI_LINKAGE_CHECK.
const (
	LinkageWarn    = "warn"
	LinkageEnforce = "enforce"
	LinkageOff     = "off"
)

// VerifyLinkage checks that the image's entrypoint binary resolves all
// of its shared libraries, including glibc symbol versions, against the
// image's own runtime base. This catches binaries compiled in a newer CI
// image than the Dockerfile's runtime base, which otherwise only fail at
// deploy time with e.g. "version `GLIBC_2.38' not found".
//
// In warn mode a failed check is printed but VerifyLinkage returns nil.
func (d *Docker) VerifyLinkage(ctx context.Context, image, mode string) error {
	switch mode {
	case LinkageOff:
		fmt.Println("skipping linkage check (CI_LINKAGE_CHECK=off)")
		return nil
	case LinkageWarn, LinkageEnforce:
	default:
		return fmt.Errorf("invalid CI_LINKAGE_CHECK=%q, expected warn, enforce, or off", mode)
	}

	inspect, _, err := d.cli.ImageInspectWithRaw(ctx, image)
	if err != nil {
		return fmt.Errorf("failed to inspect image %s: %v", image, err)
	}
	bin, ok := entrypointBinary(inspect.Config)
	if !ok {
		fmt.Println("skipping linkage check:", image, "has no Entrypoint or Cmd")
		return nil
	}

	fmt.Println("verifying", bin, "in", image, "links against the image's runtime base...")
	exitCode, err := d.runLinkageCheck(ctx, image, bin)
	if err != nil {
		return err
	}
	if exitCode == 0 {
		return nil
	}

	if mode == LinkageEnforce {
		return fmt.Errorf("linkage check failed for %s: the binary was likely built on a newer OS than the Dockerfile's runtime base", image)
	}
	fmt.Println("WARNING: linkage check failed. This image will likely fail to start in production.")
	fmt.Println("WARNING: this will fail the build once CI_LINKAGE_CHECK=enforce becomes the default.")
	return nil
}

// runLinkageCheck runs linkageCheck in a throwaway container from image
// and returns its exit code. Images without a usable /bin/sh (scratch,
// distroless) report exit code 0: they require static binaries anyway.
func (d *Docker) runLinkageCheck(ctx context.Context, image, bin string) (int64, error) {
	// Cancelled on return so the ContainerWait goroutine is released
	// even when the container never starts.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	created, err := d.cli.ContainerCreate(ctx, &container.Config{
		Image:      image,
		Entrypoint: []string{"/bin/sh", "-c", linkageCheck, "linkage-check"},
		Cmd:        []string{bin},
	}, nil, nil, nil, "")
	if err != nil {
		return 0, fmt.Errorf("failed to create linkage check container: %v", err)
	}
	defer func() {
		_ = d.cli.ContainerRemove(context.Background(), created.ID, types.ContainerRemoveOptions{Force: true})
	}()

	// Subscribe before starting so a fast exit isn't missed.
	waitCh, errCh := d.cli.ContainerWait(ctx, created.ID, container.WaitConditionNextExit)

	if err := d.cli.ContainerStart(ctx, created.ID, types.ContainerStartOptions{}); err != nil {
		if isMissingShell(err) {
			fmt.Println("skipping linkage check:", image, "has no /bin/sh")
			return 0, nil
		}
		return 0, fmt.Errorf("failed to start linkage check container: %v", err)
	}

	var exitCode int64
	select {
	case res := <-waitCh:
		if res.Error != nil {
			return 0, fmt.Errorf("linkage check container failed: %s", res.Error.Message)
		}
		exitCode = res.StatusCode
	case err := <-errCh:
		return 0, fmt.Errorf("failed waiting for linkage check container: %v", err)
	}

	logs, err := d.cli.ContainerLogs(ctx, created.ID, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		return 0, fmt.Errorf("failed to read linkage check output: %v", err)
	}
	defer logs.Close()
	if _, err := stdcopy.StdCopy(os.Stdout, os.Stdout, logs); err != nil {
		return 0, fmt.Errorf("failed to read linkage check output: %v", err)
	}

	return exitCode, nil
}

// entrypointBinary returns the binary docker would exec for the image:
// the first Entrypoint element, falling back to the first Cmd element.
func entrypointBinary(cfg *container.Config) (string, bool) {
	if cfg == nil {
		return "", false
	}
	if len(cfg.Entrypoint) > 0 && cfg.Entrypoint[0] != "" {
		return cfg.Entrypoint[0], true
	}
	if len(cfg.Cmd) > 0 && cfg.Cmd[0] != "" {
		return cfg.Cmd[0], true
	}
	return "", false
}

// isMissingShell reports whether a container start error means the
// image has no /bin/sh to run the check with.
func isMissingShell(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "/bin/sh") &&
		(strings.Contains(msg, "no such file or directory") || strings.Contains(msg, "executable file not found"))
}

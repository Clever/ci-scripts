package docker

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/moby/buildkit/frontend/dockerfile/dockerignore"
	"golang.org/x/sync/errgroup"

	"github.com/Clever/ci-scripts/internal/environment"
)

// Dockers documentation doesn't provide any examples for using their
// daemon client. This blog post is very helpful for reference:
// https://www.loginradius.com/blog/engineering/build-push-docker-images-golang/

// Docker is a wrapper around the docker Go client which wraps
// complexity in a simple API.
type Docker struct {
	cli      *client.Client
	ecrCreds map[string]types.AuthConfig
	awsCfg   aws.Config
}

// New initializes a new docker daemon client and caches ecr credentials
// for all 4 regions.
func New(ctx context.Context, ecrUploadRole string) (*Docker, error) {
	cl, err := client.NewClientWithOpts(client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize client: %v", err)
	}
	d := &Docker{
		cli:    cl,
		awsCfg: environment.AWSCfg(ctx, ecrUploadRole),
	}

	grp, ctx := errgroup.WithContext(ctx)
	d.ecrCreds = map[string]types.AuthConfig{}
	for _, r := range environment.Regions {
		r := r
		grp.Go(func() error { return d.ecrCredentials(ctx, r) })
	}

	if err := grp.Wait(); err != nil {
		return nil, err
	}

	return d, nil
}

// Build the dockerfile using the provided context dir. dockerfile can
// be a full filepath. If dockerfile is an empty string, then the
// default 'Dockerfile' name is used in the context dir.
func (d *Docker) Build(ctx context.Context, contextDir, dockerfile string, tags []string) error {
	if dockerfile == "" {
		dockerfile = "Dockerfile"
	}
	fmt.Println("building", tags, "from", dockerfile, "...")
	excludes, err := readDockerignore(contextDir)
	if err != nil {
		return fmt.Errorf("failed to read docker ignore: %v", err)
	}

	tar, err := archive.TarWithOptions(contextDir, &archive.TarOptions{
		ExcludePatterns: excludes,
	})
	if err != nil {
		return fmt.Errorf("failed to build docker context: %v", err)
	}

	res, err := d.cli.ImageBuild(ctx, tar, types.ImageBuildOptions{
		Tags:       tags,
		Dockerfile: dockerfile,
		// Removes any intermediary build images.
		Remove: true,
	})
	if err != nil {
		return fmt.Errorf("unable to build image: %v", err)
	}
	defer res.Body.Close()
	return print(res.Body)
}

// Push the tags to their private ecr repository. If a tag is not for a
// private ecr repository, Push will panic. Each tag is pushed in a
// separate goroutine.
func (d *Docker) Push(ctx context.Context, tags []string) error {
	grp, grpCtx := errgroup.WithContext(ctx)

	for _, tag := range tags {
		tag := tag
		fmt.Println("pushing", tag)
		parts := strings.Split(tag, ".")
		region := parts[3]

		grp.Go(func() error {
			// TODO: check if the repository exists, if it doesn't this just
			// nondescriptly endlessly retries.
			res, err := d.cli.ImagePush(grpCtx, tag, types.ImagePushOptions{
				RegistryAuth: encodeCreds(d.ecrCreds[region]),
			})
			if err != nil {
				return fmt.Errorf("unable to push image: %v", err)
			}

			defer res.Close()
			return print(res)
		})
	}

	return grp.Wait()
}

// fetch and cache docker client ecr credentials for the specified region.
func (d *Docker) ecrCredentials(ctx context.Context, region string) error {
	fmt.Println("fetching ecr credentials for", region, "...")
	cfg := d.awsCfg.Copy()
	cfg.Region = region

	authRes, err := ecr.NewFromConfig(cfg).GetAuthorizationToken(ctx, &ecr.GetAuthorizationTokenInput{})
	if err != nil {
		return fmt.Errorf("failed to get ecr login credentials: %v", err)
	}

	if len(authRes.AuthorizationData) != 1 {
		return fmt.Errorf("expected one authorization but got %d", len(authRes.AuthorizationData))
	}
	auth := authRes.AuthorizationData[0]

	creds, err := base64.StdEncoding.DecodeString(*auth.AuthorizationToken)
	if err != nil {
		return errors.New("failed to decode ecr auth token")
	}

	// aws GetAuthorizationToken returns the credentials in the format
	// of "<user name>:<password>". Reference:
	// https://github.com/awslabs/amazon-ecr-credential-helper/blob/main/ecr-login/api/client.go#L285
	parts := strings.SplitN(string(creds), ":", 2)
	if len(parts) < 2 {
		return fmt.Errorf("invalid token: expected two parts, got %d", len(parts))
	}

	d.ecrCreds[region] = types.AuthConfig{
		Username:      parts[0],
		Password:      parts[1],
		ServerAddress: *auth.ProxyEndpoint,
	}

	return nil
}

func encodeCreds(cfg types.AuthConfig) string {
	bs, _ := json.Marshal(cfg)
	return base64.URLEncoding.EncodeToString(bs)
}

// readDockerignore reads the .dockerignore file in the context
// directory and returns the list of paths to exclude
func readDockerignore(contextDir string) ([]string, error) {
	f, err := os.Open(filepath.Join(contextDir, ".dockerignore"))
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return []string{}, nil
	case err != nil:
		return nil, err
	}
	defer f.Close()

	return dockerignore.ReadAll(f)
}

// print writes the docker build output to stdout and parses any errors
// returned by the build daemon. If the build daemon encountered any
// build errors, an error is returned by print. If there were no build
// errors then print returns nil.
func print(r io.Reader) error {
	var line string
	scanner := bufio.NewScanner(io.TeeReader(r, os.Stdout))
	for scanner.Scan() {
		line = scanner.Text()
	}

	e := struct {
		Error string `json:"error"`
	}{}
	if err := json.Unmarshal([]byte(line), &e); err != nil {
		return fmt.Errorf("failed to unmarhsal docker daemon response: %v", err)
	}
	if e.Error != "" {
		return fmt.Errorf("error from docker daemon: %s", e.Error)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read docker daemon response: %v", err)
	}

	return nil
}

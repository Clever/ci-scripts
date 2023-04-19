# goci

`goci` is a small command line application which performs packaging and
publishing of Clever build artifacts.

# Configuration

goci does not accept any arguments and is instead configured entirely
through environment variables and launch config settings. See the
[environment](../../internal/environment/environment.go) package for
detailed documentation of environment variables for configuration. goci
reads it's configuration from the `build` section of the launch config
of each application.

# Multi-app Support

goci will automatically detect all laucn configs in the `launch`
directory, then perform the following actions as needed.

1. detect the run type of the application
2. build any docker images
3. publish all built docker images to ECFR
4. publish all pre-built lambdas to s3

Currently, goci does not build lambdas.

## Running and testing locally

Since goci is just a go app, it can be run locally after building the
binary. goci expects to be run from within the root of an applications
repo. You may use the `testApp` within this repo for testing various
launch configurations of goci.

```bash
$ go build -o ./bin/goci ./cmd/goci

$ cp ./bin/goci ./testApp

$ cd testApp && goci
```

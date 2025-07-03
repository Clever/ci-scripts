# goci

`goci` is a small command line application which performs building, publishing, and deploying applications. It is the next evolution of many of the scripts in this repository, and offers more automation, configurability and optimization.

## Configuration

goci accepts very limited arguments which merely change the mode it runs in. The rest of the configuration is entirely through environment variables and launch config settings. See the[environment](../../internal/environment/environment.go) package for detailed documentation of environment variables for configuration. goci reads it's configuration from the `build` section of the launch config of each application. See the [build section](https://github.com/Clever/catapult/blob/master/swagger.yml#L1773) of the launch yaml to learn about the various parameters which configure goci.

## Modes

1. `goci detect` detects any changed applications according to their launch configuration. This can be used to pass a name of apps to another script.
2. `goci artifact-build-publish-deploy` builds, publishes and deploys any application artifacts.
3. `goci validate` validates an applications go version, while also checking for compatible branch naming conventions for catapult.
4. `goci publish-utility` publishes catalog-info.yaml to the service catalog.


## Multi-app Support

goci will automatically detect all launch configs in the `launch`
directory, then perform the following actions as needed.

1. detect the run type of the application
2. Run any configured build commands
3. build any docker images
4. publish all built docker images to all ECR regions
5. publish all lambdas to s3 in all regions.
6. Sync all changed apps with catalog config
7. Publish new application versions to catapult
8. Deploy any changed applications.

## Development

Since goci is just a go app, it can be run locally after building the binary. goci expects to be run from within the root of an applications repo.

There is a [test app](./testApp) in this repository which you can use to run goci on locally and test various build configurations. Additionally, you can leverage the built in integrations tests in https://github.com/Clever/circleci-orbs/tree/master, which call goci to test new builds of goci in CI.

### Running locally
Running goci locally is not easy because it leverages a lot of variables set in our CI environments. Generally it would be easier to use the private orbs repo, or make a branch on another app like catapult and pull in the development branch of goci there for testing in CI. If you need to run locally, you can view each of the variables required for local runs in the environment file and provide them all to the command.

```bash
go build -o ./bin/goci ./cmd/goci

cp ./bin/goci ./testApp

cd testApp && goci
```

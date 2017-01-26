# ci-scripts

Re-usable continuous integration (CI) scripts

Inspired by: https://circleci.com/blog/continuous-integration-at-segment/

Owned by `#eng-infra`.

## Scripts

### Docker

Logs into Docker registry, then builds and pushes docker image.
Docker image is tagged with 7 character git commit SHA.

```
$ ./circleci/docker-publish [DOCKER_USER] [DOCKER_PASS] [DOCKER_EMAIL] [ORG]
```

### NPM Publish

Authenticates to NPM and publishes a package.

```
$ ./circleci/npm-publish [NPM_TOKEN] [PACKAGE_DIR]
```

### Github Release

Publishes content from `[ARTIFACTS_DIR]` as a Github Release.

```
$ ./circleci/github-release [GITHUB_TOKEN] [ARTIFACTS_DIR]
```

### Catapult

Publishes your application and build in [catapult](github.com/clever/catapult).

```
$ ./circleci/catapult-publish [CATAPULT_URL] [CATAPULT_USER] [CATAPULT_PASS] [APP_NAME]
```

If you need to publish multiple applications, run this command once for each.

### Report-card

Runs [report-card](github.com/clever/report-card).

```
$ ./circleci/report-card [DOCKER_USER] [DOCKER_PASS] [DOCKER_EMAIL] [GITHUB_TOKEN]
```

### Mongo install

Installs a specific Mongo version, rather than the default version in CircleCI.
Default version in Ubuntu 14.04 (Trusty) image is [here](https://circleci.com/docs/build-image-trusty/#mongodb). (`v3.0.7` at time of writing)

```
$ ./circleci/mongo-install [VERSION]
```

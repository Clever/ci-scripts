# ci-scripts

Re-usable continuous integration (CI) scripts

Inspired by: https://circleci.com/blog/continuous-integration-at-segment/

Owned by `#eng-infra`.

## Scripts

### Docker

```
$ ./circleci/docker-publish <docker_user> <docker_pass> <docker_email> <org>
```

Publishes an image to DockerHub at `<org>/<repo>`.
Tags the image according to the git short commit sha; this is 7 characters long and equal to `$(git rev-parse --short HEAD)`.

### Npm

```
$ ./circleci/npm-publish <npm_token> <package_dir>
```

Publish package to NPM, according to configuration specified in `<package_dir>/package.json`.
Requires `npm_token` to authenticate.

### Github Release

_Not yet implemented._

```
$ ./circleci/github-release <version> <path_to_artifacts>
```

### Catapult

_Not yet implemented._

Publishes your application and build in [catapult](github.com/clever/catapult).

```
$ ./circleci/catapult <catapult_url> <catapult_app>
```

If you need to publish multiple applications, run this command once for each.

### Report-card

Runs [report-card](github.com/clever/report-card).

```
$ ./circleci/report-card <docker_user> <docker_pass> <docker_email>
```

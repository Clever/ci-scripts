# ci-scripts

Re-usable continuous integration (CI) scripts

Inspired by: https://circleci.com/blog/continuous-integration-at-segment/

Owned by `#eng-infra`.

## Scripts

### Docker

```
$ ./circleci-docker [user] [pass] [email] [org]
```

Publishes an image to DockerHub at `<org>/<repo>`.
Tags the image according to the git short commit sha; this is 7 characters long and equal to `$(git rev-parse --short HEAD)`.

### Npm

```
$ ./circleci-npm
```

Publish package to NPM, according to configuration specified in `package.json`.

### Github Release

```
$ ./circleci-gh-release [version] [path_to_artifacts]
```

### Catapult

Publishes your application and build in [catapult](github.com/clever/catapult).

```
$ ./circleci-catapult $catapult_url $catapult_app
```

If you need to publish multiple applications, run this command once for each.

### Report-card

Runs [report-card](github.com/clever/report-card).

```
$ ./circleci-report-card
```

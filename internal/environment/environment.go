package environment

import (
	"fmt"
	"os"
	"strconv"
)

var (
	// ECRAccountID is the account ID for clever's ECR repositories.
	ECRAccountID = envMustString("ECR_ACCOUNT_ID", true)
	// FullSHA1 is the full git commit SHA being built in CI.
	FullSHA1 = envMustString("CIRCLE_SHA1", true)
	// ShortSHA1 is the first 7 characters of the git commit SHA being
	// built in CI.
	ShortSHA1 = FullSHA1[:7]
	// ECRAccessKeyID is the AWS access key ID which has correct
	// permissions to upload images to ECR.
	ECRAccessKeyID = envMustString("ECR_PUSH_ID", false)
	// ECRSecretAccessKey is the AWS secret key which has correct
	// permissions to upload images to ECR.
	ECRSecretAccessKey = envMustString("ECR_PUSH_SECRET", false)
	// LambdaArtifactBucketPrefix is the prefix of the S3 buckets which
	// hold Clever's lambda artifacts. There are 4 total – one for each
	// region. The naming scheme is '<prefix>-<region>'
	LambdaArtifactBucketPrefix = envMustString("LAMBDA_AWS_BUCKET", true)
	// CatapultURL is the dns of the circle-ci-integrations ALB
	// including the protocol.
	CatapultURL = envMustString("CATAPULT_URL", true)
	// User the name of the GitHub user who triggered
	// the CI build
	User = envMustString("CIRCLE_PROJECT_USERNAME", true)
	// Repo is the name of the repo being built in this
	// CI run.
	Repo = envMustString("CIRCLE_PROJECT_REPONAME", true)
	// CircleBuildNum is the CI build number.
	CircleBuildNum = envMustInt64("CIRCLE_BUILD_NUM", true)
	// Branch is the git branch being built in CI.
	Branch = envMustString("CIRCLE_BRANCH", true)

	// Regions is the set of regions this app should perform
	// operations in.
	Regions = []string{"us-west-1", "us-west-2", "us-east-1", "us-east-2"}

	// Local is a boolean which should be set to true when running
	// locally on a developers machine.
	Local = os.Getenv("LOCAL") == "true"
)

func envMustString(key string, localRequired bool) string {
	v := os.Getenv(key)
	if v == "" && localRequired {
		fmt.Println("env variable missing:", key)
		os.Exit(1)
	}
	return v
}

func envMustInt64(key string, localRequired bool) int64 {
	v := os.Getenv(key)
	if v == "" && localRequired {
		fmt.Println("env variable missing:", key)
		os.Exit(1)
	}

	i, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		fmt.Println(fmt.Errorf("invalid value %s cannot be converted to int64: %v", v, err))
		os.Exit(1)
	}
	return i
}

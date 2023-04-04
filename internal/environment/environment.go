package environment

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
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
	// LambdaAccessKeyID is the AWS access key ID which has correct
	// permissions to upload to S3 lambda artifact buckets.
	LambdaAccessKeyID = envMustString("LAMBDA_AWS_ACCESS_KEY_ID", true)
	// LambdaSecretAccessKey is the AWS secret key which has correct
	// permissions to upload to S3 lambda artifact buckets.
	LambdaSecretAccessKey = envMustString("LAMBDA_AWS_SECRET_ACCESS_KEY", true)
	// CatapultUser is the username to access circle-ci-integrations via
	// basic auth.
	CatapultUser = envMustString("CATAPULT_USER", true)
	// CatapultPassword is the password to access circle-ci-integrations
	// via basic auth.
	CatapultPassword = envMustString("CATAPULT_PASS", true)
	// CircleUser is the name of the ci user assigned by our ci
	// environment.
	CircleUser = envMustString("CIRCLE_PROJECT_USERNAME", true)
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

// AWSCfg initializes an AWS config or exits with code 0 on failure. If
// this app is run locally, then this function automatically pulls
// config from the default credential chain which can be populated with
// saml2aws. If not run locally, then the passed in id and secret key
// are used with a static credentials provider.
func AWSCfg(ctx context.Context, accessKeyID, secretKey string) aws.Config {
	opts := []func(*config.LoadOptions) error{
		config.WithRegion("us-west-1"),
	}

	// In local environment we use the default credentials chain that
	// will automatically pull creds from saml2aws,
	if !Local {
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKeyID, secretKey, ""),
		))
	}

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		fmt.Println("failed to load aws config:", err)
		os.Exit(1)
	}

	return cfg
}

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

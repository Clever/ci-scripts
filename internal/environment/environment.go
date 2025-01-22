package environment

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

var (
	// ECRAccountID is the account ID for clever's ECR repositories.
	ECRAccountID = envMustString("ECR_ACCOUNT_ID", true)
	// FullSHA1 is the full git commit SHA being built in CI.
	FullSHA1 = envMustString("CIRCLE_SHA1", true)
	// ShortSHA1 is the first 7 characters of the git commit SHA being
	// built in CI.
	ShortSHA1 = FullSHA1[:7]
	// LambdaArtifactBucketPrefix is the prefix of the S3 buckets which
	// hold Clever's lambda artifacts. There are 4 total – one for each
	// region. The naming scheme is '<prefix>-<region>'
	LambdaArtifactBucketPrefix = envMustString("LAMBDA_AWS_BUCKET", true)
	// PreviousPipelineCompare is the git commit range to run change
	// detection commands against when running for the primary branch.
	PreviousPipelineCompare = envMustString("PREVIOUS_PIPELINE_COMPARE", false)
	// PrimaryCompare is the git commit range to run change detection
	// commands against when running for a non-primary branch.
	PrimaryCompare = envMustString("MASTER_COMPARE", true)
	// CatapultURL is the dns of the circle-ci-integrations ALB
	// including the protocol.
	CatapultURL = envMustString("CATAPULT_URL", true)
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
	// OidcLambdaRole is the ARN of the role used to assume the lambda
	// publishing role.
	OidcLambdaRole = envMustString("OIDC_LAMBDA_ROLE", false)
	// OidcEcrUploadRole is the ARN of the role used to assume the ecr
	// upload role.
	OidcEcrUploadRole = envMustString("OIDC_ECR_UPLOAD_ROLE", false)
	// circleOidcTokenV2 is the oidc token used to assume roles in CI.
	// It is provided by circle-ci.
	circleOidcTokenV2 = envMustString("CIRCLE_OIDC_TOKEN_V2", false)

	// Regions is the set of regions this app should perform
	// operations in.
	Regions = []string{"us-west-1", "us-west-2", "us-east-1"}

	// Local is a boolean which should be set to true when running
	// locally on a developers machine.
	Local = os.Getenv("LOCAL") == "true"
)

// AWS doesn't provide a way to get the token from a string so we will
// use this to satisfy the interface.
type tokenRetriever struct{}

func (tokenRetriever) GetIdentityToken() ([]byte, error) {
	return []byte(circleOidcTokenV2), nil
}

// AWSCfg initializes an AWS config or exits with code 0 on failure. If
// this app is run locally, then this function automatically pulls
// config from the default credential chain which can be populated with
// saml2aws. If not run locally, then the passed role and profile are
// used with oidc in circle ci.
func AWSCfg(ctx context.Context, oidcRole string) aws.Config {
	opts := []func(*config.LoadOptions) error{
		config.WithRegion("us-west-2"),
	}

	// In local environment we use the default credentials chain that
	// will automatically pull creds from saml2aws,
	if !Local {
		stsCfg, err := config.LoadDefaultConfig(ctx, opts...)
		if err != nil {
			fmt.Println("failed to load aws sts config:", err)
			os.Exit(1)
		}

		opts = append(opts, config.WithCredentialsProvider(
			stscreds.NewWebIdentityRoleProvider(
				sts.NewFromConfig(stsCfg),
				oidcRole,
				tokenRetriever{},
				func(o *stscreds.WebIdentityRoleOptions) {
					o.RoleSessionName = "oidc-goci-role-session"
				},
			),
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
	} else if Local && !localRequired && v == "" {
		return 0
	}

	i, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		fmt.Println(fmt.Errorf("invalid value %s cannot be converted to int64: %v", v, err))
		os.Exit(1)
	}
	return i
}

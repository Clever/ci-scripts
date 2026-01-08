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
	ecrAccountID = ""
	// FullSHA1 is the full git commit SHA being built in CI.
	fullSHA1 = ""
	// ShortSHA1 is the first 7 characters of the git commit SHA being
	// built in CI.
	shortSHA1 = ""
	// LambdaArtifactBucketPrefix is the prefix of the S3 buckets which
	// hold Clever's lambda artifacts. There are 4 total – one for each
	// region. The naming scheme is '<prefix>-<region>'
	lambdaArtifactBucketPrefix = ""
	// PreviousPipelineCompare is the git commit range to run change
	// detection commands against when running for the primary branch.
	previousPipelineCompare = ""
	// PrimaryCompare is the git commit range to run change detection
	// commands against when running for a non-primary branch.
	primaryCompare = ""
	// CatapultURL is the dns of the circle-ci-integrations ALB
	// including the protocol.
	catapultURL = ""
	// CatapultUser is the username to access circle-ci-integrations via
	// basic auth.
	catapultUser = ""
	// CatapultPassword is the password to access circle-ci-integrations
	// via basic auth.
	catapultPassword = ""
	// CircleUser is the name of the ci user assigned by our ci
	// environment.
	circleUser = ""
	// Repo is the name of the repo being built in this
	// CI run.
	repo = ""
	// CircleBuildNum is the CI build number.
	circleBuildNum = int64(0)
	// Branch is the git branch being built in CI.
	branch = ""
	// OidcLambdaRole is the ARN of the role used to assume the lambda
	// publishing role.
	oidcLambdaRole = ""
	// OidcEcrUploadRole is the ARN of the role used to assume the ecr
	// upload role.
	oidcEcrUploadRole = ""

	// LambdaRegions is the set of regions to upload Lambda artifacts to.
	// Lambda artifacts are not replicated and must be uploaded to each region.
	LambdaRegions = []string{"us-west-1", "us-west-2", "us-east-1"}

	// Local is a boolean which should be set to true when running
	// locally on a developers machine.
	Local = os.Getenv("LOCAL") == "true"
)

func ECRAccountID() string {
	if ecrAccountID == "" {
		ecrAccountID = envMustString("ECR_ACCOUNT_ID", true)
	}
	return ecrAccountID
}

func FullSHA1() string {
	if fullSHA1 == "" {
		fullSHA1 = envMustString("CIRCLE_SHA1", true)
	}
	return fullSHA1
}

func ShortSHA1() string {
	if shortSHA1 == "" {
		shortSHA1 = FullSHA1()[:7]
	}
	return shortSHA1
}

func LambdaArtifactBucketPrefix() string {
	if lambdaArtifactBucketPrefix == "" {
		lambdaArtifactBucketPrefix = envMustString("LAMBDA_AWS_BUCKET", true)
	}
	return lambdaArtifactBucketPrefix
}

func PreviousPipelineCompare() string {
	if previousPipelineCompare == "" {
		previousPipelineCompare = envMustString("PREVIOUS_PIPELINE_COMPARE", false)
	}
	return previousPipelineCompare
}

func PrimaryCompare() string {
	if primaryCompare == "" {
		primaryCompare = envMustString("MASTER_COMPARE", true)
	}
	return primaryCompare
}

func CatapultURL() string {
	if catapultURL == "" {
		catapultURL = envMustString("CATAPULT_URL", true)
	}
	return catapultURL
}

func CatapultUser() string {
	if catapultUser == "" {
		catapultUser = envMustString("CATAPULT_USER", true)
	}
	return catapultUser
}

func CatapultPassword() string {
	if catapultPassword == "" {
		catapultPassword = envMustString("CATAPULT_PASS", true)
	}
	return catapultPassword
}

func CircleUser() string {
	if circleUser == "" {
		circleUser = envMustString("CIRCLE_PROJECT_USERNAME", true)
	}
	return circleUser
}

func Repo() string {
	if repo == "" {
		repo = envMustString("CIRCLE_PROJECT_REPONAME", true)
	}
	return repo
}

func CircleBuildNum() int64 {
	if circleBuildNum == 0 {
		circleBuildNum = envMustInt64("CIRCLE_BUILD_NUM", true)
	}
	return circleBuildNum
}

func Branch() string {
	if branch == "" {
		branch = envMustString("CIRCLE_BRANCH", true)
	}
	return branch
}

func OidcLambdaRole() string {
	if oidcLambdaRole == "" {
		oidcLambdaRole = envMustString("OIDC_LAMBDA_ROLE", false)
	}
	return oidcLambdaRole
}

func OidcEcrUploadRole() string {
	if oidcEcrUploadRole == "" {
		oidcEcrUploadRole = envMustString("OIDC_ECR_UPLOAD_ROLE", false)
	}
	return oidcEcrUploadRole
}

// AWS doesn't provide a way to get the token from a string so we will
// use this to satisfy the interface.
type tokenRetriever struct{}

func (tokenRetriever) GetIdentityToken() ([]byte, error) {
	return []byte(envMustString("CIRCLE_OIDC_TOKEN_V2", false)), nil
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

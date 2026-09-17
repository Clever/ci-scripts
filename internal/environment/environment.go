package environment

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

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
	// OidcEventBridgeRole is the ARN of the role used to publish platform
	// events to EventBridge.
	oidcEventBridgeRole = ""

	// LambdaRegions is the set of regions to upload Lambda artifacts to.
	// Lambda artifacts are not replicated and must be uploaded to each region.
	LambdaRegions = []string{"us-west-1", "us-west-2", "us-east-1"}

	// Local is a boolean which should be set to true when running
	// locally on a developers machine.
	Local = os.Getenv("LOCAL") == "true"
	// SlingshotURL is the DNS of the slingshot service.
	slingshotURL = ""

	// CircleTriggeredBy is the username of the user who triggered the CI run.
	circleTriggeredBy = ""
	// URL for circle-ci-integrations
	ciIntegrationsURL = ""
	// username for basic authentication to circle-ci-integrations
	ciIntegrationsUser = ""
	// password for basic authentication to circle-ci-integrations
	ciIntegrationsPassword = ""
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

func CircleTriggeredBy() string {
	if circleTriggeredBy == "" {
		circleTriggeredBy = envMustString("CIRCLE_USERNAME", true)
	}
	return circleTriggeredBy
}

// FleetAutomationUser is the who-is-who service-account identity we attribute a
// fleet campaign merge's deploy to. Fleet stamps every campaign merge commit
// with a clever-fleet[bot] co-author trailer (see isFleetMerge), and this
// identity is registered in who-is-who so downstream deploy services (Slingshot,
// dapple) resolve it. See INFRA-1076.
const FleetAutomationUser = "clever-fleet[bot]"

// fleetMergeCoAuthorLogin is the GitHub App login fleet stamps as a co-author on
// every campaign merge commit (matched case-insensitively).
const fleetMergeCoAuthorLogin = "clever-fleet[bot]"

var (
	fleetMergeChecked bool
	fleetMergeResult  bool
)

// DeployUser returns the identity to attribute a deploy to. A fleet campaign
// merge is performed by automation — the CircleCI trigger user is a bot (e.g.
// backstage-clever[bot], which does the bulk merge), not a person — so
// CIRCLE_USERNAME never resolves in who-is-who. We positively detect a fleet
// merge by the clever-fleet[bot] co-author trailer that fleet stamps on every
// campaign merge commit, and attribute those deploys to the fleet service
// account. Every other pipeline keeps CIRCLE_USERNAME and its existing hard-fail
// on an empty/unresolvable trigger, so a non-fleet automation merge is never
// misattributed to fleet. See INFRA-1076.
func DeployUser() string {
	if isFleetMerge() {
		return FleetAutomationUser
	}
	return CircleTriggeredBy()
}

// isFleetMerge reports whether HEAD was produced by a fleet campaign merge,
// detected by a clever-fleet[bot] co-author trailer on the commit message. The
// result is memoized; any git error (not a checkout, etc.) is treated as "not a
// fleet merge" so we fall back to CIRCLE_USERNAME.
func isFleetMerge() bool {
	if !fleetMergeChecked {
		fleetMergeResult = detectFleetMerge()
		fleetMergeChecked = true
	}
	return fleetMergeResult
}

func detectFleetMerge() bool {
	out, err := exec.Command("git", "log", "-1", "--format=%B", "HEAD").Output()
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(out), "\n") {
		l := strings.ToLower(strings.TrimSpace(line))
		if strings.HasPrefix(l, "co-authored-by:") && strings.Contains(l, fleetMergeCoAuthorLogin) {
			return true
		}
	}
	return false
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

func OidcEventBridgeRole() string {
	if oidcEventBridgeRole == "" {
		oidcEventBridgeRole = envMustString("OIDC_EVENTBRIDGE_ROLE", false)
	}
	return oidcEventBridgeRole
}

func SlingshotURL() string {
	if slingshotURL == "" {
		slingshotURL = envMustString("SLINGSHOT_URL", true)
	}
	return slingshotURL
}

func CIIntegrationsUrl() string {
	if ciIntegrationsURL == "" {
		ciIntegrationsURL = envMustString("CIRCLE_CI_INTEGRATIONS_URL", true)
	}
	return ciIntegrationsURL
}

func CIIntegrationsUser() string {
	if ciIntegrationsUser == "" {
		ciIntegrationsUser = envMustString("CIRCLE_CI_INTEGRATIONS_USER", true)
	}
	return ciIntegrationsUser
}

func CIIntegrationsPassword() string {
	if ciIntegrationsPassword == "" {
		ciIntegrationsPassword = envMustString("CIRCLE_CI_INTEGRATIONS_PASS", true)
	}
	return ciIntegrationsPassword
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

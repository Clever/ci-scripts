package environment

import (
	"fmt"
	"os"
)

var (
	ECRAccountID         = envMustString("ECR_ACCOUNT_ID", true)
	ShortSHA1            = envMustString("CIRCLE_SHA1", true)[:7]
	ECRAccessKeyID       = envMustString("ECR_PUSH_ID", false)
	ECRSecretAccessKey   = envMustString("ECR_PUSH_SECRET", false)
	LambdaArtifactBucket = envMustString("LAMBDA_AWS_BUCKET", true)
	CatapultURL          = envMustString("CATAPULT_URL", true)

	AWSRegions = []string{"us-west-1", "us-west-2", "us-east-1", "us-east-2"}

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

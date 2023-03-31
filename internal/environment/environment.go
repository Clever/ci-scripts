package environment

import (
	"fmt"
	"os"
)

var (
	ECRAccountID         = envMustString("ECR_ACCOUNT_ID")
	ShortSHA1            = envMustString("CIRCLE_SHA1")[:7]
	ECRAccessKeyID       = envMustString("ECR_PUSH_ID")
	ECRSecretAccessKey   = envMustString("ECR_PUSH_SECRET")
	LambdaArtifactBucket = envMustString("LAMBDA_AWS_BUCKET")
	CatapultURL          = envMustString("CATAPULT_URL")

	AWSRegions = []string{"us-west-1", "us-west-2", "us-east-1", "us-east-2"}

	Local = os.Getenv("LOCAL") == "true"
)

func envMustString(key string) string {
	v := os.Getenv(key)
	if v == "" {
		fmt.Println("env variable missing:", key)
		os.Exit(1)
	}
	return v
}

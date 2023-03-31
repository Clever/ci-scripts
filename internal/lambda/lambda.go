package lambda

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/Clever/ci-scripts/internal/environment"
)

// Lambda wraps s3 to provide a simple API building and publishing lambdas.
type Lambda struct {
	awsCfg aws.Config
}

// New initializes a new Lambda handling wrapper with it's s3 client.
func New(cfg aws.Config) *Lambda {
	return &Lambda{awsCfg: cfg}
}

// Publish an already built binary archive to s3 using the artifact
// names as the key.
func (l *Lambda) Publish(ctx context.Context, binaryPath, artifactName, region string) error {
	bucket := fmt.Sprintf("%s-%s", environment.LambdaArtifactBucket, region)
	key := fmt.Sprintf("%[1]s/%[2]s/%[1]s.zip", artifactName, environment.ShortSHA1)
	s3uri := fmt.Sprintf("s3://%s/%s", bucket, key)

	fmt.Println("uploading lambda binary", binaryPath, "to", s3uri, "...")

	f, err := os.Open(binaryPath)
	if err != nil {
		return fmt.Errorf("unable to open binary file %s: %v", binaryPath, err)
	}

	cfg := l.awsCfg.Copy()
	cfg.Region = region

	_, err = s3.NewFromConfig(cfg).PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   f,
	})
	if err != nil {
		return fmt.Errorf("failed to upload %s to %s: %v", binaryPath, s3uri, err)
	}

	return nil
}

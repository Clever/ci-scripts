package lambda

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"golang.org/x/sync/errgroup"

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

// Publish an already built lambda artifact archive to s3 using the
// artifact name as the key. The archive is pushed to each of the 4 aws
// regions. Each region is pushed in it's own goroutine.
func (l *Lambda) Publish(ctx context.Context, binaryPath, artifactName string) error {
	grp, grpCtx := errgroup.WithContext(ctx)
	for _, region := range environment.Regions {
		region := region
		bucket := fmt.Sprintf("%s-%s", environment.LambdaArtifactBucketPrefix, region)
		key := s3Key(artifactName)
		s3uri := fmt.Sprintf("s3://%s/%s", bucket, key)

		fmt.Println("uploading lambda artifact", binaryPath, "to", s3uri, "...")

		grp.Go(func() error {
			f, err := os.Open(binaryPath)
			if err != nil {
				return fmt.Errorf("unable to open lambda artifact archive %s: %v", binaryPath, err)
			}
			cfg := l.awsCfg.Copy()
			cfg.Region = region

			res, err := s3.NewFromConfig(cfg).PutObject(grpCtx, &s3.PutObjectInput{
				Bucket: aws.String(bucket),
				Key:    aws.String(key),
				Body:   f,
			})
			if err != nil {
				return fmt.Errorf("failed to upload %s to %s: %v", binaryPath, s3uri, err)
			}
			fmt.Printf("res: %s - %s\n", *res.VersionId, *res.ETag)
			return nil
		})
	}

	return grp.Wait()
}

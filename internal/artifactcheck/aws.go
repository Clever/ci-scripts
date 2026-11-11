package artifactcheck

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	ecrtypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/Clever/ci-scripts/internal/environment"
)

type ecrImageChecker struct {
	client *ecr.Client
}

type s3ObjectChecker struct {
	client *s3.Client
}

func NewAWSCheckers(ctx context.Context) (ImageChecker, ObjectChecker) {
	config := environment.AWSCfg(ctx, environment.OidcArtifactReadRole())
	return &ecrImageChecker{client: ecr.NewFromConfig(config)}, &s3ObjectChecker{client: s3.NewFromConfig(config)}
}

func (checker *ecrImageChecker) ImageExists(ctx context.Context, repository, tag string) (bool, error) {
	_, err := checker.client.DescribeImages(ctx, &ecr.DescribeImagesInput{
		RegistryId:     aws.String(environment.ECRAccountID()),
		RepositoryName: aws.String(repository),
		ImageIds:       []ecrtypes.ImageIdentifier{{ImageTag: aws.String(tag)}},
	})
	var imageNotFound *ecrtypes.ImageNotFoundException
	var repositoryNotFound *ecrtypes.RepositoryNotFoundException
	if errors.As(err, &imageNotFound) || errors.As(err, &repositoryNotFound) {
		return false, nil
	}
	return err == nil, err
}

func (checker *s3ObjectChecker) ObjectExists(ctx context.Context, bucket, key string) (bool, error) {
	_, err := checker.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	var notFound *s3types.NotFound
	if errors.As(err, &notFound) {
		return false, nil
	}
	return err == nil, err
}

package cmd

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Basics struct {
	Client  *s3.Client
	Context context.Context
}

func NewS3Basics(ctx context.Context, awsConfig aws.Config) *S3Basics {
	return &S3Basics{
		Context: ctx,
		Client:  s3.NewFromConfig(awsConfig),
	}
}

func (basics *S3Basics) GetListObjectsPaginator(config partitionConfig) *s3.ListObjectsV2Paginator {
	params := &s3.ListObjectsV2Input{
		Bucket: &config.S3Bucket,
	}

	if len(config.S3Prefix) != 0 {
		params.Prefix = &config.S3Prefix
	}

	if len(config.S3ObjectDelimiter) != 0 {
		params.Delimiter = &config.S3ObjectDelimiter
	}

	// Create the Paginator for the ListObjectsV2 operation.
	paginator := s3.NewListObjectsV2Paginator(basics.Client, params, func(o *s3.ListObjectsV2PaginatorOptions) {
		if config.S3MaxKeys != 0 {
			o.Limit = config.S3MaxKeys
		}
	})

	return paginator
}

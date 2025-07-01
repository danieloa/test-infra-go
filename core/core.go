package core

import (
	"fmt"

	conf "github.com/danieloa/test-infra-go/config"

	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/s3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

var (
	fmBuckets             = []string{"rox-conf", "rox-state"}
	generalPurposeBuckets = []string{"backups", "logs", "general-logs"}
)

func SetupEKSInfrastructure(ctx *pulumi.Context, config *conf.Config, tags pulumi.StringMap) error {
	// Placeholder for core infrastructure setup logic
	// This could include setting up EKS, RDS, S3, etc.
	fmt.Println("Setting up core infrastructure...")

	// Initialize EKS cluster
	platform, err := setupEKSCluster(ctx, config, "platform-", tags)
	if err != nil {
		return fmt.Errorf("failed to set up EKS cluster: %w", err)
	}
	// Initialize tekton cluster
	tekton, err := setupEKSCluster(ctx, config, "tekton-", tags)
	if err != nil {
		return fmt.Errorf("failed to set up tekton cluster: %w", err)
	}

	// Export cluster information for other stages
	ctx.Export("platformClusterId", platform.ID())
	ctx.Export("platformClusterEndpoint", platform.Endpoint)
	ctx.Export("tektonClusterId", tekton.ID())
	ctx.Export("tektonClusterEndpoint", tekton.Endpoint)

	return nil
}

func SetupS3Buckets(ctx *pulumi.Context, config *conf.Config, tags pulumi.StringMap) error {
	productDomain := config.ProductDomain

	// General purpose buckets (no CORS, private)
	for _, bucketName := range generalPurposeBuckets {
		fullName := bucketName + productDomain
		bucket, err := createS3BucketUnified(ctx, fullName, tags, false, pulumi.String(s3.CannedAclPrivate), nil)
		if err != nil {
			return fmt.Errorf("failed to create S3 bucket %s: %w", fullName, err)
		}
		ctx.Export(fmt.Sprintf("%sBucketId", bucketName), bucket.ID())
		ctx.Export(fmt.Sprintf("%sBucketName", bucketName), bucket.Bucket)
		ctx.Export(fmt.Sprintf("%sBucketArn", bucketName), bucket.Arn)
		ctx.Export(fmt.Sprintf("%sBucketDomainName", bucketName), bucket.BucketDomainName)
	}

	// FM buckets (with CORS, public-read)
	for _, fmBucketName := range fmBuckets {
		fullName := fmBucketName + productDomain
		corsOrigins := []string{"https://" + fullName}
		bucket, err := createS3BucketUnified(ctx, fullName, tags, true, pulumi.String(s3.CannedAclPublicRead), corsOrigins)
		if err != nil {
			return fmt.Errorf("failed to create S3 bucket %s: %w", fullName, err)
		}
		ctx.Export(fmt.Sprintf("%sBucketId", fmBucketName), bucket.ID())
		ctx.Export(fmt.Sprintf("%sBucketName", fmBucketName), bucket.Bucket)
		ctx.Export(fmt.Sprintf("%sBucketArn", fmBucketName), bucket.Arn)
		ctx.Export(fmt.Sprintf("%sBucketDomainName", fmBucketName), bucket.BucketDomainName)
	}

	return nil
}

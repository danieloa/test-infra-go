package core

import (
	"fmt"

	conf "github.com/danieloa/test-infra-go/config"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

var (
	publicBuckets                = []string{"images", "assets", "public-data"}
	generalPurposePrivateBuckets = []string{"backups", "logs", "general-logs"}
)

func SetupEKSInfrastructure(ctx *pulumi.Context, config *conf.Config, tags pulumi.StringMap) error {
	ctx.Log.Info("Setting up core infrastructure...", nil)

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

	// Private buckets (with encryption, no CORS, restricted access)
	for _, bucketName := range generalPurposePrivateBuckets {
		fullName := bucketName + "-" + productDomain
		bucket, err := createS3BucketUnified(ctx, fullName, tags, false, false, nil)
		if err != nil {
			return fmt.Errorf("failed to create private S3 bucket %s: %w", fullName, err)
		}
		ctx.Export(fmt.Sprintf("%sBucketId", bucketName), bucket.ID())
		ctx.Export(fmt.Sprintf("%sBucketName", bucketName), bucket.Bucket)
		ctx.Export(fmt.Sprintf("%sBucketArn", bucketName), bucket.Arn)
		ctx.Export(fmt.Sprintf("%sBucketDomainName", bucketName), bucket.BucketDomainName)
	}

	// Public buckets (with CORS, public-read) - intentionally public for web content
	for _, publicBucketName := range publicBuckets {
		fullName := publicBucketName + "-" + productDomain
		corsOrigins := []string{"https://" + fullName}
		bucket, err := createS3BucketUnified(ctx, fullName, tags, true, true, corsOrigins)
		if err != nil {
			return fmt.Errorf("failed to create public S3 bucket %s: %w", fullName, err)
		}
		ctx.Export(fmt.Sprintf("%sBucketId", publicBucketName), bucket.ID())
		ctx.Export(fmt.Sprintf("%sBucketName", publicBucketName), bucket.Bucket)
		ctx.Export(fmt.Sprintf("%sBucketArn", publicBucketName), bucket.Arn)
		ctx.Export(fmt.Sprintf("%sBucketDomainName", publicBucketName), bucket.BucketDomainName)
	}

	ctx.Log.Info(fmt.Sprintf("Created %d private and %d public S3 buckets", len(generalPurposePrivateBuckets), len(publicBuckets)), nil)
	return nil
}

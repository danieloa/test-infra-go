package core

import (
	"fmt"
	"regexp"

	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/s3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// validateS3BucketName validates that the bucket name meets S3 naming requirements
func validateS3BucketName(name string) error {
	if len(name) < 3 || len(name) > 63 {
		return fmt.Errorf("bucket name must be between 3 and 63 characters long")
	}
	// S3 bucket naming rules: lowercase letters, numbers, dots, hyphens
	// Must start and end with a letter or number
	match, _ := regexp.MatchString(`^[a-z0-9][a-z0-9.-]*[a-z0-9]$`, name)
	if !match {
		return fmt.Errorf("bucket name must start and end with a lowercase letter or number and contain only lowercase letters, numbers, dots, and hyphens")
	}
	// Cannot have consecutive periods
	if regexp.MustCompile(`\.\.`).MatchString(name) {
		return fmt.Errorf("bucket name cannot contain consecutive periods")
	}
	return nil
}

// createS3BucketUnified creates an S3 bucket with encryption, versioning, lifecycle rules, and optional CORS.
// Parameters:
//   - withCors: if true, adds CORS rules
//   - isPublic: if true, allows public read access (use with caution!)
//   - corsOrigins: list of allowed CORS origins
func createS3BucketUnified(ctx *pulumi.Context, bucketName string, tags pulumi.StringMap, withCors bool, isPublic bool, corsOrigins []string) (*s3.Bucket, error) {
	// Validate bucket name
	if err := validateS3BucketName(bucketName); err != nil {
		return nil, fmt.Errorf("invalid bucket name '%s': %w", bucketName, err)
	}

	// Create bucket with basic configuration
	args := &s3.BucketArgs{
		Bucket: pulumi.String(bucketName),
		// ACL is deprecated - using BucketPublicAccessBlock instead
		LifecycleRules: s3.BucketLifecycleRuleArray{
			&s3.BucketLifecycleRuleArgs{
				Id:      pulumi.String("default"),
				Enabled: pulumi.Bool(true),
				NoncurrentVersionTransitions: s3.BucketLifecycleRuleNoncurrentVersionTransitionArray{
					&s3.BucketLifecycleRuleNoncurrentVersionTransitionArgs{
						Days:         pulumi.Int(90),
						StorageClass: pulumi.String("STANDARD_IA"),
					},
					&s3.BucketLifecycleRuleNoncurrentVersionTransitionArgs{
						Days:         pulumi.Int(180),
						StorageClass: pulumi.String("GLACIER"),
					},
				},
				NoncurrentVersionExpiration: &s3.BucketLifecycleRuleNoncurrentVersionExpirationArgs{
					Days: pulumi.Int(270),
				},
			},
		},
		Versioning: &s3.BucketVersioningArgs{
			Enabled: pulumi.Bool(true),
		},
		ServerSideEncryptionConfiguration: &s3.BucketServerSideEncryptionConfigurationArgs{
			Rule: &s3.BucketServerSideEncryptionConfigurationRuleArgs{
				ApplyServerSideEncryptionByDefault: &s3.BucketServerSideEncryptionConfigurationRuleApplyServerSideEncryptionByDefaultArgs{
					SseAlgorithm: pulumi.String("AES256"),
				},
				BucketKeyEnabled: pulumi.Bool(true),
			},
		},
		Tags: tags,
	}

	if withCors && len(corsOrigins) > 0 {
		args.CorsRules = s3.BucketCorsRuleArray{
			&s3.BucketCorsRuleArgs{
				AllowedHeaders: pulumi.StringArray{pulumi.String("*")},
				AllowedMethods: pulumi.StringArray{
					pulumi.String("GET"),
					pulumi.String("PUT"),
					pulumi.String("POST"),
				},
				AllowedOrigins: func() pulumi.StringArray {
					arr := make(pulumi.StringArray, len(corsOrigins))
					for i, o := range corsOrigins {
						arr[i] = pulumi.String(o)
					}
					return arr
				}(),
				ExposeHeaders: pulumi.StringArray{pulumi.String("ETag")},
				MaxAgeSeconds: pulumi.Int(3000),
			},
		}
	}

	bucket, err := s3.NewBucket(ctx, bucketName, args)
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 bucket %s: %w", bucketName, err)
	}

	// Configure bucket ownership controls
	_, err = s3.NewBucketOwnershipControls(ctx, bucketName+"-ownership", &s3.BucketOwnershipControlsArgs{
		Bucket: bucket.ID(),
		Rule: &s3.BucketOwnershipControlsRuleArgs{
			ObjectOwnership: pulumi.String("BucketOwnerPreferred"),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create bucket ownership controls for %s: %w", bucketName, err)
	}

	// Configure public access block (block all public access for private buckets)
	if !isPublic {
		_, err = s3.NewBucketPublicAccessBlock(ctx, bucketName+"-public-access-block", &s3.BucketPublicAccessBlockArgs{
			Bucket:                bucket.ID(),
			BlockPublicAcls:       pulumi.Bool(true),
			BlockPublicPolicy:     pulumi.Bool(true),
			IgnorePublicAcls:      pulumi.Bool(true),
			RestrictPublicBuckets: pulumi.Bool(true),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create public access block for %s: %w", bucketName, err)
		}
	}

	return bucket, nil
}

package core

import (
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/s3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// createS3BucketUnified creates an S3 bucket with optional CORS and ACL settings.
func createS3BucketUnified(ctx *pulumi.Context, bucketName string, tags pulumi.StringMap, withCors bool, acl pulumi.StringInput, corsOrigins []string) (*s3.Bucket, error) {
	args := &s3.BucketArgs{
		Bucket: pulumi.String(bucketName),
		Acl:    acl,
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
		Tags: tags,
	}

	if withCors && len(corsOrigins) > 0 {
		args.CorsRules = s3.BucketCorsRuleArray{
			&s3.BucketCorsRuleArgs{
				AllowedHeaders: pulumi.StringArray{pulumi.String("*")},
				AllowedMethods: pulumi.StringArray{
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

	_, err = s3.NewBucketOwnershipControls(ctx, bucketName+"-ownership", &s3.BucketOwnershipControlsArgs{
		Bucket: bucket.ID(),
		Rule: &s3.BucketOwnershipControlsRuleArgs{
			ObjectOwnership: pulumi.String("BucketOwnerPreferred"),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create bucket ownership controls for %s: %w", bucketName, err)
	}

	return bucket, nil
}

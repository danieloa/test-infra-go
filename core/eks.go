package core

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	conf "github.com/danieloa/test-infra-go/config"
	"github.com/danieloa/test-infra-go/stages"

	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/eks"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/iam"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func getEksAssumeRoleJSON() ([]byte, error) {
	tmpJSON1, err := json.Marshal(map[string]interface{}{
		"Version": "2012-10-17",
		"Statement": []map[string]interface{}{
			map[string]interface{}{
				"Action": []string{
					"sts:AssumeRole",
					"sts:TagSession",
				},
				"Effect": "Allow",
				"Principal": map[string]interface{}{
					"Service": "eks.amazonaws.com",
				},
			},
		},
	})
	return tmpJSON1, err
}

func getEc2AssumeRoleJSON() ([]byte, error) {
	tmpJSON0, err := json.Marshal(map[string]interface{}{
		"Version": "2012-10-17",
		"Statement": []map[string]interface{}{
			map[string]interface{}{
				"Action": []string{
					"sts:AssumeRole",
				},
				"Effect": "Allow",
				"Principal": map[string]interface{}{
					"Service": "ec2.amazonaws.com",
				},
			},
		},
	})
	return tmpJSON0, err
}

func getSubnetIds(ctx *pulumi.Context, config *conf.Config, subnetType string) (pulumi.StringArrayOutput, error) {
	if len(config.Azs) == 0 {
		return pulumi.StringArrayOutput{}, fmt.Errorf("no availability zones specified in config")
	}

	region, _ := ctx.GetConfig("config:region")
	if region == "" {
		region, _ = ctx.GetConfig("aws:region") // Check if region is set in Pulumi config
		if region == "" {
			// read from os environment variable
			region = os.Getenv("AWS_REGION")
		}
	}

	// Stack naming format: customer-stage-stack-region
	// For VPC stage with VPC stack: danieloa-vpc-vpc-us-east-1
	var stackName = fmt.Sprintf("%s-%s-%s-%s", config.Customer, string(stages.StageVPC), string(stages.StackVPC), region)
	vpcRef, err := pulumi.NewStackReference(ctx, stackName, nil)
	if err != nil {
		return pulumi.StringArrayOutput{}, fmt.Errorf("failed to create stack reference: %w", err)
	}

	subnetIds := vpcRef.GetOutput(pulumi.String(subnetType)).ApplyT(func(ids []string) pulumi.StringArray {
		if len(ids) == 0 {
			return nil
		}
		subnetArray := make(pulumi.StringArray, len(ids))
		for i, id := range ids {
			subnetArray[i] = pulumi.String(id)
		}
		return subnetArray
	}).(pulumi.StringArrayOutput)

	return subnetIds, nil
}

func setupEKSCluster(ctx *pulumi.Context, config *conf.Config, prefix string, tags pulumi.StringMap) (*eks.Cluster, error) {

	ctx.Log.Info(fmt.Sprintf("Setting up EKS cluster with prefix: %s", prefix), nil)

	tmpJSON0, err := getEc2AssumeRoleJSON()
	if err != nil {
		return nil, err
	}

	ec2AssumeRolePolicy := string(tmpJSON0)
	node, err := iam.NewRole(ctx, prefix+"node", &iam.RoleArgs{
		Name:             pulumi.String(prefix + "eks-auto-node"),
		AssumeRolePolicy: pulumi.String(ec2AssumeRolePolicy),
		Tags:             tags,
	})
	if err != nil {
		return nil, err
	}

	tmpJSON1, err := getEksAssumeRoleJSON()
	if err != nil {
		return nil, err
	}
	json1 := string(tmpJSON1)
	cluster, err := iam.NewRole(ctx, prefix+"cluster", &iam.RoleArgs{
		Name:             pulumi.String(prefix + "cluster"),
		AssumeRolePolicy: pulumi.String(json1),
		Tags:             tags,
	})
	if err != nil {
		return nil, err
	}
	clusterAmazonEKSClusterPolicy, err := iam.NewRolePolicyAttachment(ctx, prefix+"cluster_AmazonEKSClusterPolicy", &iam.RolePolicyAttachmentArgs{
		PolicyArn: pulumi.String("arn:aws:iam::aws:policy/AmazonEKSClusterPolicy"),
		Role:      cluster.Name,
	})
	if err != nil {
		return nil, err
	}
	clusterAmazonEKSComputePolicy, err := iam.NewRolePolicyAttachment(ctx, prefix+"cluster_AmazonEKSComputePolicy", &iam.RolePolicyAttachmentArgs{
		PolicyArn: pulumi.String("arn:aws:iam::aws:policy/AmazonEKSComputePolicy"),
		Role:      cluster.Name,
	})
	if err != nil {
		return nil, err
	}
	clusterAmazonEKSBlockStoragePolicy, err := iam.NewRolePolicyAttachment(ctx, prefix+"cluster_AmazonEKSBlockStoragePolicy", &iam.RolePolicyAttachmentArgs{
		PolicyArn: pulumi.String("arn:aws:iam::aws:policy/AmazonEKSBlockStoragePolicy"),
		Role:      cluster.Name,
	})
	if err != nil {
		return nil, err
	}
	clusterAmazonEKSLoadBalancingPolicy, err := iam.NewRolePolicyAttachment(ctx, prefix+"cluster_AmazonEKSLoadBalancingPolicy", &iam.RolePolicyAttachmentArgs{
		PolicyArn: pulumi.String("arn:aws:iam::aws:policy/AmazonEKSLoadBalancingPolicy"),
		Role:      cluster.Name,
	})
	if err != nil {
		return nil, err
	}
	clusterAmazonEKSNetworkingPolicy, err := iam.NewRolePolicyAttachment(ctx, prefix+"cluster_AmazonEKSNetworkingPolicy", &iam.RolePolicyAttachmentArgs{
		PolicyArn: pulumi.String("arn:aws:iam::aws:policy/AmazonEKSNetworkingPolicy"),
		Role:      cluster.Name,
	})
	if err != nil {
		return nil, err
	}

	eksName := prefix[:len(prefix)-1] // remove trailing dash
	if eksName == "" {
		eksName = "platform"
	}

	snetIds, err := getSubnetIds(ctx, config, eksName+"SubnetIds")
	if err != nil {
		return nil, fmt.Errorf("failed to get availability zone IDs: %w", err)
	}

	eksCluster, err := eks.NewCluster(ctx, eksName, &eks.ClusterArgs{
		Name: pulumi.String(eksName),
		AccessConfig: &eks.ClusterAccessConfigArgs{
			AuthenticationMode: pulumi.String("API"),
		},
		RoleArn:                    cluster.Arn,
		Version:                    pulumi.String("1.31"),
		BootstrapSelfManagedAddons: pulumi.Bool(false),
		ComputeConfig: &eks.ClusterComputeConfigArgs{
			Enabled: pulumi.Bool(true),
			NodePools: pulumi.StringArray{
				pulumi.String("general-purpose"),
			},
			NodeRoleArn: node.Arn,
		},
		KubernetesNetworkConfig: &eks.ClusterKubernetesNetworkConfigArgs{
			ElasticLoadBalancing: &eks.ClusterKubernetesNetworkConfigElasticLoadBalancingArgs{
				Enabled: pulumi.Bool(true),
			},
		},
		StorageConfig: &eks.ClusterStorageConfigArgs{
			BlockStorage: &eks.ClusterStorageConfigBlockStorageArgs{
				Enabled: pulumi.Bool(true),
			},
		},
		VpcConfig: &eks.ClusterVpcConfigArgs{
			EndpointPrivateAccess: pulumi.Bool(true),
			EndpointPublicAccess:  pulumi.Bool(true),
			SubnetIds:             snetIds,
		},
	}, pulumi.DependsOn([]pulumi.Resource{
		clusterAmazonEKSClusterPolicy,
		clusterAmazonEKSComputePolicy,
		clusterAmazonEKSBlockStoragePolicy,
		clusterAmazonEKSLoadBalancingPolicy,
		clusterAmazonEKSNetworkingPolicy,
	}))
	if err != nil {
		return nil, err
	}
	// Attach additional node policies for CloudWatch logs and other common operations
	nodePolicies := []string{
		"arn:aws:iam::aws:policy/AmazonEKSWorkerNodeMinimalPolicy",
		"arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryPullOnly",
		"arn:aws:iam::aws:policy/CloudWatchAgentServerPolicy",
	}

	for _, policyArn := range nodePolicies {
		policyName := prefix + "node_" + policyArn[strings.LastIndex(policyArn, "/")+1:]
		_, err = iam.NewRolePolicyAttachment(ctx, policyName, &iam.RolePolicyAttachmentArgs{
			PolicyArn: pulumi.String(policyArn),
			Role:      node.Name,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to attach policy %s: %w", policyArn, err)
		}
	}

	ctx.Log.Info(fmt.Sprintf("Successfully created EKS cluster: %s", eksName), nil)
	return eksCluster, nil
}

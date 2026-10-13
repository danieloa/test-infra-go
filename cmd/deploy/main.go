package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	aws "github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/smithy-go"

	danieloaConfig "github.com/danieloa/test-infra-go/config"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type CLIConfig struct {
	ConfigFile string
	Stage      string
	Action     string
}

func main() {
	var cfg CLIConfig

	var rootCmd = &cobra.Command{
		Use:   "deploy [stage]",
		Short: "danieloa Platform Infrastructure Deployment Tool",
		Long:  `A smart CLI tool for staged deployment of danieloa Platform infrastructure using Pulumi.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg.Stage = args[0]
			return deploy(cfg)
		},
	}

	var initCmd = &cobra.Command{
		Use:   "init",
		Short: "Initialize S3 backend",
		RunE: func(cmd *cobra.Command, args []string) error {
			return initBackend(cfg)
		},
	}

	// Add flags to root command
	rootCmd.Flags().StringVarP(&cfg.ConfigFile, "config", "c", "deployment-configurations/us-east-1_dev.yaml", "Config file path")
	rootCmd.Flags().StringVarP(&cfg.Action, "action", "a", "preview", "Pulumi action: up, down, preview, refresh")

	// Add flags to init command
	initCmd.Flags().StringVarP(&cfg.ConfigFile, "config", "c", "deployment-configurations/us-east-1_dev.yaml", "Config file path")

	rootCmd.AddCommand(initCmd)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func loadConfig(filename string) (*danieloaConfig.Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config danieloaConfig.Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

func checkAWSAuth(region string) error {
	ctx := context.Background()
	var opts []func(*awsConfig.LoadOptions) error
	if region != "" {
		opts = append(opts, awsConfig.WithRegion(region))
	}
	cfg, err := awsConfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := sts.NewFromConfig(cfg)
	identity, err := client.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		profile := os.Getenv("AWS_PROFILE")
		if profile != "" {
			fmt.Printf("❌ AWS authentication failed. Please run: aws sso login --profile %s\n", profile)
		}
		return fmt.Errorf("AWS authentication failed: %w", err)
	}

	profileInfo := ""
	if p := os.Getenv("AWS_PROFILE"); p != "" {
		profileInfo = fmt.Sprintf(" (profile: %s)", p)
	}
	fmt.Printf("✅ AWS authentication successful%s (Account: %s, ARN: %s)\n",
		profileInfo, aws.ToString(identity.Account), aws.ToString(identity.Arn))
	return nil
}

func initBackend(cfg CLIConfig) error {
	config, err := loadConfig(cfg.ConfigFile)
	if err != nil {
		return err
	}

	customerName := config.GetCustomerName()

	// Deploy to each region in config
	region := os.Getenv("AWS_REGION")

	fmt.Printf("🚀 Initializing S3 backend for customer: %s in primary region: %s\n", customerName, region)

	if err := checkAWSAuth(region); err != nil {
		return err
	}

	// Create single S3 bucket in primary region for all stacks
	s3Bucket := fmt.Sprintf("%s-%s-pulumi-state", config.Customer, config.Environment)

	fmt.Printf("📦 Creating S3 bucket: %s\n", s3Bucket)

	if err := createS3BucketIfNotExists(s3Bucket, region); err != nil {
		return err
	}

	// Configure Pulumi to use S3 backend
	s3BackendURL := fmt.Sprintf("s3://%s", s3Bucket)
	loginCmd := exec.Command("pulumi", "login", s3BackendURL)
	if err := loginCmd.Run(); err != nil {
		return fmt.Errorf("failed to configure Pulumi S3 backend: %w", err)
	}

	fmt.Printf("✅ S3 backend initialized: %s\n", s3BackendURL)
	fmt.Printf("   All region stacks will use this single state bucket\n")

	return nil
}

func createS3BucketIfNotExists(bucketName, region string) error {
	ctx := context.Background()

	// Load AWS SDK config
	cfg, err := awsConfig.LoadDefaultConfig(ctx, awsConfig.WithRegion(region))
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client
	client := s3.NewFromConfig(cfg)

	// Check if bucket exists
	_, err = client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err == nil {
		fmt.Printf("✅ S3 bucket already exists: %s\n", bucketName)
		return nil
	}

	fmt.Printf("📦 Creating S3 bucket: %s\n", bucketName)

	// Create bucket with appropriate configuration
	createBucketInput := &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	}

	// For regions other than us-east-1, we need to specify LocationConstraint
	if region != "us-east-1" {
		createBucketInput.CreateBucketConfiguration = &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint(region),
		}
	}

	_, err = client.CreateBucket(ctx, createBucketInput)
	if err != nil {
		// Check if error is because bucket already exists
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			switch apiErr.ErrorCode() {
			case "BucketAlreadyExists", "BucketAlreadyOwnedByYou":
				fmt.Printf("✅ S3 bucket already exists: %s\n", bucketName)
				return nil
			}
		}
		return fmt.Errorf("failed to create S3 bucket: %w", err)
	}

	// Enable versioning
	_, err = client.PutBucketVersioning(ctx, &s3.PutBucketVersioningInput{
		Bucket: aws.String(bucketName),
		VersioningConfiguration: &types.VersioningConfiguration{
			Status: types.BucketVersioningStatusEnabled,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to enable versioning: %w", err)
	}

	fmt.Printf("✅ S3 bucket created successfully: %s\n", bucketName)

	return nil
}

func deploy(cfg CLIConfig) error {
	config, err := loadConfig(cfg.ConfigFile)
	if err != nil {
		return err
	}

	// Validate stage
	validStages := []string{"vpc", "core", "apps"}
	isValidStage := false
	for _, stage := range validStages {
		if cfg.Stage == stage {
			isValidStage = true
			break
		}
	}
	if !isValidStage {
		return fmt.Errorf("invalid stage '%s'. Valid stages: %v", cfg.Stage, validStages)
	}

	fmt.Printf("🚀 danieloa Platform Infrastructure Deployment\n")
	fmt.Printf("   Customer: %s\n", config.GetCustomerName())
	fmt.Printf("   Stage: %s\n", cfg.Stage)
	fmt.Printf("   Action: %s\n", cfg.Action)
	fmt.Printf("\n")

	// Set environment variables
	os.Setenv("CONFIG_FILE", cfg.ConfigFile)
	fmt.Printf("Using config file: %s\n", cfg.ConfigFile)

	// Deploy to each region in config
	region := os.Getenv("AWS_REGION")

	fmt.Printf("🌎 Processing region: %s\n", region)

	if err := checkAWSAuth(region); err != nil {
		return err
	}

	stacksPerStage := map[string][]string{
		"vpc":  {"acm", "r53", "vpc"},
		"core": {"eks", "s3", "rds", "secrets", "kms"},
		"apps": {"app1", "app2", "app3"},
	}

	for _, stack := range stacksPerStage[cfg.Stage] {
		// Stack naming format: customer-stage-stack-region
		// Example: danieloa-vpc-vpc-us-east-1
		stackName := fmt.Sprintf("%s-%s-%s-%s", config.GetCustomerName(), cfg.Stage, stack, region)
		fmt.Printf("🌎 stackName : %s\n", stackName)

		if err := deployToRegion(cfg, stackName, region); err != nil {
			return fmt.Errorf("deployment failed for region %s: %w", region, err)
		}

		fmt.Printf("✅ Region %s completed successfully!\n\n", region)

		// Show completion message
		switch cfg.Action {
		case "preview", "diff":
			fmt.Printf("✅ %s completed successfully for all regions!\n", cfg.Action)
		case "up":
			fmt.Printf("✅ %s stage deployed successfully to all regions!\n", cfg.Stage)
			// Show next steps
			switch cfg.Stage {
			case "vpc":
				fmt.Printf("\n🔗 Next: Deploy core stage with: deploy core\n")
			case "core":
				fmt.Printf("\n🔗 Next: Deploy apps stage with: deploy apps\n")
			case "apps":
				fmt.Printf("\n🎉 All stages deployed! Your infrastructure is ready.\n")
			}
		case "down":
			fmt.Printf("✅ %s stage destroyed successfully in all regions!\n", cfg.Stage)
		case "refresh":
			fmt.Printf("✅ Infrastructure state refreshed successfully in all regions!\n")
		}
	}

	return nil
}

func deployToRegion(cfg CLIConfig, stackName string, region string) error {
	// Select/create stack
	stackCmd := exec.Command("pulumi", "stack", "select", stackName, "--create")
	if err := stackCmd.Run(); err != nil {
		return fmt.Errorf("failed to select/create stack %s: %w", stackName, err)
	}

	// Set configuration values for this region
	// Extract stack identifier from full stack name
	// stackName format is: customer-stage-stack-region (e.g., danieloa-vpc-vpc-us-east-1)
	// We need to extract just the stack part (3rd component)
	stackParts := strings.Split(stackName, "-")
	stackIdentifier := ""
	// Stack format: customer-stage-stack-region
	// For danieloa-vpc-vpc-us-east-1: parts are [danieloa, vpc, vpc, us, east, 1]
	// We want the 3rd element (index 2)
	if len(stackParts) >= 3 {
		stackIdentifier = stackParts[2]
	}

	configs := map[string]string{
		"stage":      cfg.Stage,
		"stack":      stackIdentifier,
		"aws:region": region,
		"configFile": cfg.ConfigFile,
	}

	for key, value := range configs {
		configCmd := exec.Command("pulumi", "config", "set", key, value)
		if err := configCmd.Run(); err != nil {
			return fmt.Errorf("failed to set config %s: %w", key, err)
		}
	}

	// Deploy based on action
	var deployCmd *exec.Cmd
	switch cfg.Action {
	case "preview":
		deployCmd = exec.Command("pulumi", "preview", "--diff")
	case "up":
		deployCmd = exec.Command("pulumi", "up", "--yes")
	case "down":
		deployCmd = exec.Command("pulumi", "destroy", "--yes")
	case "refresh":
		deployCmd = exec.Command("pulumi", "refresh", "--yes")
	default:
		return fmt.Errorf("unsupported action: %s", cfg.Action)
	}

	deployCmd.Stdout = os.Stdout
	deployCmd.Stderr = os.Stderr
	deployCmd.Env = os.Environ()

	return deployCmd.Run()
}
